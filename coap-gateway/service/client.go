package service

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	grpcClient "github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	pbCQRS "github.com/plgd-dev/cloud/resource-aggregate/pb"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	kitSync "github.com/plgd-dev/kit/sync"
	"github.com/plgd-dev/sdk/schema"
	"github.com/plgd-dev/sdk/schema/cloud"
)

type observedResource struct {
	href        string
	observation *tcp.Observation
}

type authCtx struct {
	pbCQRS.AuthorizationContext
	AccessToken string
	UserID      string
	Expire      time.Time
}

const pendingDeviceSubscriptionToken = "pending"

//Client a setup of connection
type Client struct {
	server   *Server
	coapConn *tcp.ClientConn

	observedResources     map[string]map[int64]observedResource // [deviceID][instanceID]
	observedResourcesLock sync.Mutex

	resourceSubscriptions *kitSync.Map // [token]
	deviceSubscriptions   *kitSync.Map // [token]

	mutex   sync.Mutex
	authCtx authCtx
}

//newClient create and initialize client
func newClient(server *Server, client *tcp.ClientConn) *Client {
	return &Client{
		server:                server,
		coapConn:              client,
		observedResources:     make(map[string]map[int64]observedResource),
		resourceSubscriptions: kitSync.NewMap(),
		deviceSubscriptions:   kitSync.NewMap(),
	}
}

func ToClient(v interface{}, ok bool) (*Client, bool) {
	if !ok {
		return nil, false
	}
	if v == nil {
		return nil, false
	}
	c, ok := v.(*Client)
	return c, ok
}

func (client *Client) remoteAddrString() string {
	return client.coapConn.RemoteAddr().String()
}

func (client *Client) Context() context.Context {
	return client.coapConn.Context()
}

func (client *Client) cancelResourceSubscription(token string, wantWait bool) (bool, error) {
	s, ok := grpcClient.ToResourceSubscription(client.resourceSubscriptions.PullOut(token))
	if !ok {
		return false, nil
	}
	wait, err := s.Cancel()
	if err != nil {
		return false, err
	}
	if wantWait {
		wait()
	}
	return true, nil
}

func (client *Client) observeResource(ctx context.Context, deviceID, href string, observable, allowDuplicit bool) (err error) {
	log.Debugf("coap-gw: client.observeResource /%v%v ins %v: observe resource", deviceID, href)
	instanceID := getInstanceID(href)
	client.observedResourcesLock.Lock()
	defer client.observedResourcesLock.Unlock()
	if _, ok := client.observedResources[deviceID]; !ok {
		client.observedResources[deviceID] = make(map[int64]observedResource)
	}
	if _, ok := client.observedResources[deviceID][instanceID]; ok {
		if allowDuplicit {
			return nil
		}
		return fmt.Errorf("resource is already already published")
	}
	return client.addObservedResourceLocked(ctx, deviceID, href, observable)
}

func (client *Client) getResourceContent(ctx context.Context, deviceID, href string) {
	resp, err := client.coapConn.Get(ctx, href)
	if err != nil {
		log.Errorf("cannot get resource /%v%v content: %v", deviceID, href, err)
		return
	}
	defer pool.ReleaseMessage(resp)
	err = client.notifyContentChanged(deviceID, href, resp)
	if err != nil {
		// cloud is unsynchronized against device. To recover cloud state, client need to reconnect to cloud.
		log.Errorf("cannot get resource /%v%v content: %v", deviceID, href, err)
		client.Close()
	}
	if resp.Code() == coapCodes.NotFound {
		client.unpublishResources(ctx, []pbRA.ResourceId{pbRA.ResourceId{DeviceId: deviceID, Href: href}})
	}
}

func (client *Client) addObservedResourceLocked(ctx context.Context, deviceID, href string, observale bool) error {
	var observation *tcp.Observation
	obs := observale
	if href == cloud.StatusHref {
		return nil
	}
	instanceID := getInstanceID(href)

	if obs {
		obs, err := client.coapConn.Observe(ctx, href, func(req *pool.Message) {
			err := client.notifyContentChanged(deviceID, href, req)
			if err != nil {
				// cloud is unsynchronized against device. To recover cloud state, client need to reconnect to cloud.
				log.Errorf("cannot observe resource /%v%v: %v", deviceID, href, err)
				client.Close()
			}
			if req.Code() == coapCodes.NotFound {
				client.unpublishResources(req.Context(), []pbRA.ResourceId{pbRA.ResourceId{DeviceId: deviceID, Href: href}})
			}
		})
		if err != nil {
			log.Errorf("cannot observe resource /%v%v: %v", deviceID, href, err)
		} else {
			observation = obs
		}
	} else {
		go client.getResourceContent(ctx, deviceID, href)
	}
	client.observedResources[deviceID][instanceID] = observedResource{href: href, observation: observation}
	return nil
}

func (client *Client) getObservedResources(deviceID string, instanceIDs []int64) []pbRA.ResourceId {
	client.observedResourcesLock.Lock()
	defer client.observedResourcesLock.Unlock()

	getAllDeviceIDMatches := len(instanceIDs) == 0
	matches := make([]pbRA.ResourceId, 0, 16)

	if deviceResourcesMap, ok := client.observedResources[deviceID]; ok {
		if getAllDeviceIDMatches {
			for _, value := range deviceResourcesMap {
				matches = append(matches, pbRA.ResourceId{
					DeviceId: deviceID,
					Href:     value.href,
				})
			}
		} else {
			for _, instanceID := range instanceIDs {
				if resource, ok := deviceResourcesMap[instanceID]; ok {
					matches = append(matches, pbRA.ResourceId{
						DeviceId: deviceID,
						Href:     resource.href,
					})
				}
			}
		}
	}

	return matches
}

func (client *Client) removeResource(deviceID string, instanceID int64) {
	if device, ok := client.observedResources[deviceID]; ok {
		delete(device, instanceID)
		if len(client.observedResources[deviceID]) == 0 {
			delete(client.observedResources, deviceID)
		}
	}
}

func (client *Client) popObservation(deviceID string, instanceID int64) *tcp.Observation {
	log.Debugf("remove published resource ocf://%v/%v", deviceID, instanceID)

	var obs *tcp.Observation
	if device, ok := client.observedResources[deviceID]; ok {
		if res, ok := device[instanceID]; ok {
			obs = res.observation
			res.observation = nil
		}
	}

	return obs
}

func (client *Client) unobserveResources(ctx context.Context, resourceIDs []pbRA.ResourceId, rscsUnpublished map[string]bool) {
	observartions := client.unobserveAndRemoveResources(resourceIDs, rscsUnpublished)
	for _, obs := range observartions {
		obs.Cancel(ctx)
	}
}

// Close closes coap connection
func (client *Client) Close() error {
	err := client.coapConn.Close()
	if err != nil {
		return fmt.Errorf("cannot close client: %w", err)
	}
	return nil
}

func (client *Client) unobserveAndRemoveResources(resourceIDs []pbRA.ResourceId, rscsUnpublished map[string]bool) []*tcp.Observation {
	observartions := make([]*tcp.Observation, 0, 32)

	client.observedResourcesLock.Lock()
	defer client.observedResourcesLock.Unlock()

	for _, resourceID := range resourceIDs {
		if del, ok := rscsUnpublished[resourceID.GetHref()]; ok && del {
			log.Debugf("delete resource /%v%v", resourceID.GetDeviceId(), resourceID.GetHref())
		} else {
			log.Debugf("unobserve resource /%v%v", resourceID.GetDeviceId(), resourceID.GetHref())
		}
		var instanceID int64
		var deviceID string
		for devID, devs := range client.observedResources {
			for insID, r := range devs {
				if r.href == resourceID.GetHref() {
					instanceID = insID
					deviceID = devID
					break
				}
			}
		}

		obs := client.popObservation(deviceID, instanceID)
		if obs != nil {
			observartions = append(observartions, obs)
		}
		if rscsUnpublished[resourceID.GetHref()] {
			client.removeResource(deviceID, instanceID)
		}
	}
	return observartions
}

func (client *Client) popObservedResources() []*tcp.Observation {
	observartions := make([]*tcp.Observation, 0, 32)
	client.observedResourcesLock.Lock()
	defer client.observedResourcesLock.Unlock()
	for deviceID, instanceIDs := range client.observedResources {
		for instanceID := range instanceIDs {
			obs := client.popObservation(deviceID, instanceID)
			if obs != nil {
				observartions = append(observartions, obs)
			}
			client.removeResource(deviceID, instanceID)
		}
	}
	return observartions
}

// cleanObservedResources remove all device pbRA observation requested by cloud.
func (client *Client) cleanObservedResources() {
	for _, obs := range client.popObservedResources() {
		obs.Cancel(client.coapConn.Context())
	}
}

func (client *Client) cancelResourceSubscriptions(wantWait bool) {
	resourceSubscriptions := client.resourceSubscriptions.PullOutAll()
	for _, v := range resourceSubscriptions {
		o, ok := grpcClient.ToResourceSubscription(v, true)
		if !ok {
			continue
		}
		wait, err := o.Cancel()
		if err != nil {
			log.Errorf("cannot cancel resource subscription: %v", err)
		} else if wantWait {
			wait()
		}
	}
}

func (client *Client) cancelDeviceSubscriptions(wantWait bool) {
	deviceSubscriptions := client.deviceSubscriptions.PullOutAll()
	for _, v := range deviceSubscriptions {
		o, ok := grpcClient.ToDeviceSubscription(v.(*atomic.Value).Load(), true)
		if !ok {
			continue
		}
		wait, err := o.Cancel()
		if err != nil {
			log.Errorf("cannot cancel device subscription: %v", err)
		} else if wantWait {
			wait()
		}
	}
}

// OnClose action when coap connection was closed.
func (client *Client) OnClose() {
	log.Debugf("close client %v", client.coapConn.RemoteAddr())

	client.server.oicPingCache.Delete(client.remoteAddrString())
	client.cleanObservedResources()
	client.cancelResourceSubscriptions(false)
	client.cancelDeviceSubscriptions(false)

	oldAuthCtx := client.replaceAuthorizationContext(authCtx{})

	if oldAuthCtx.DeviceId != "" {
		client.server.expirationClientCache.Delete(oldAuthCtx.DeviceId)
		ctx, cancel := context.WithTimeout(context.Background(), client.server.RequestTimeout)
		defer cancel()
		token, err := client.server.oauthMgr.GetToken(ctx)
		if err != nil {
			log.Errorf("DeviceId %v: cannot handle sign out: cannot update cloud device status: %v", oldAuthCtx.DeviceId, err)
			return
		}
		err = client.UpdateCloudDeviceStatus(kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(ctx, token.AccessToken), oldAuthCtx.UserID), oldAuthCtx.DeviceId, oldAuthCtx.AuthorizationContext, false)
		if err != nil {
			// Device will be still reported as online and it can fix his state by next calls online, offline commands.
			log.Errorf("DeviceId %v: cannot handle sign out: cannot update cloud device status: %v", oldAuthCtx.DeviceId, err)
		}
	}
}

func (client *Client) replaceAuthorizationContext(authCtx authCtx) (oldDeviceID authCtx) {
	log.Debugf("Authorization context replaced for client %v, device %v, user %v", client.coapConn.RemoteAddr(), authCtx.GetDeviceId(), authCtx.UserID)
	client.mutex.Lock()
	defer client.mutex.Unlock()
	oldAuthContext := client.authCtx
	client.authCtx = authCtx
	return oldAuthContext
}

func (client *Client) loadAuthorizationContext() authCtx {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	return client.authCtx
}

func (client *Client) notifyContentChanged(deviceID string, href string, notification *pool.Message) error {
	authCtx := client.loadAuthorizationContext()
	if isExpired(authCtx.Expire) {
		return fmt.Errorf("cannot notify resource /%v%v content changed: token is expired", deviceID, href)
	}
	decodeMsgToDebug(client, notification, "RECEIVED-NOTIFICATION")
	ctx, err := client.server.ServiceRequestContext(authCtx.UserID)
	if err != nil {
		return fmt.Errorf("cannot notify resource /%v%v content changed: %w", deviceID, href, err)
	}
	request := coapconv.MakeNotifyResourceChangedRequest(deviceID, href, authCtx.AuthorizationContext, client.remoteAddrString(), notification)
	_, err = client.server.raClient.NotifyResourceChanged(ctx, &request)
	if err != nil {
		return fmt.Errorf("cannot notify resource /%v%v content changed: %w", deviceID, href, err)
	}
	return nil
}

func (client *Client) sendErrorConfirmResourceUpdate(deviceID, href, userID, correlationID string, authCtx pbCQRS.AuthorizationContext, code codes.Code, errToSend error) {
	ctx, err := client.server.ServiceRequestContext(userID)
	if err != nil {
		log.Errorf("cannot send error via confirm resource update: %v", err)
		return
	}

	resp := pool.AcquireMessage(ctx)
	defer pool.ReleaseMessage(resp)
	resp.SetContentFormat(message.TextPlain)
	resp.SetBody(bytes.NewReader([]byte(errToSend.Error())))
	resp.SetCode(code)
	request := coapconv.MakeConfirmResourceUpdateRequest(deviceID, href, correlationID, authCtx, client.remoteAddrString(), resp)
	_, err = client.server.raClient.ConfirmResourceUpdate(ctx, &request)
	if err != nil {
		log.Errorf("cannot send error via confirm resource update: %v", err)
	}
}

func (client *Client) updateResource(ctx context.Context, event *pb.Event_ResourceUpdatePending) error {
	authCtx := client.loadAuthorizationContext()
	if isExpired(authCtx.Expire) {
		err := fmt.Errorf("cannot update resource /%v%v: token is expired", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref())
		client.sendErrorConfirmResourceUpdate(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.UserID, event.GetCorrelationId(), authCtx.AuthorizationContext, codes.Forbidden, err)
		client.Close()
		return err
	}
	if event.GetResourceId().GetHref() == cloud.StatusHref {
		authCtx := client.loadAuthorizationContext()
		msg := pool.AcquireMessage(ctx)
		msg.SetCode(coapCodes.MethodNotAllowed)
		msg.SetSequence(client.coapConn.Sequence())
		defer pool.ReleaseMessage(msg)
		sendConfirmCtx, err := client.server.ServiceRequestContext(authCtx.UserID)
		if err != nil {
			return err
		}
		request := coapconv.MakeConfirmResourceUpdateRequest(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), event.GetCorrelationId(), authCtx.AuthorizationContext, client.remoteAddrString(), msg)
		_, err = client.server.raClient.ConfirmResourceUpdate(sendConfirmCtx, &request)
		if err != nil {
			return err
		}
		return nil
	}

	coapCtx, cancel := context.WithTimeout(ctx, client.server.RequestTimeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceUpdateRequest(coapCtx, event)
	if err != nil {
		client.sendErrorConfirmResourceUpdate(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.UserID, event.GetCorrelationId(), authCtx.AuthorizationContext, codes.BadRequest, err)
		return err
	}
	defer pool.ReleaseMessage(req)

	decodeMsgToDebug(client, req, "RESOURCE-UPDATE-REQUEST")

	resp, err := client.coapConn.Do(req)
	if err != nil {
		client.sendErrorConfirmResourceUpdate(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.UserID, event.GetCorrelationId(), authCtx.AuthorizationContext, codes.ServiceUnavailable, err)
		return err
	}
	defer pool.ReleaseMessage(resp)

	decodeMsgToDebug(client, resp, "RESOURCE-UPDATE-RESPONSE")

	if resp.Code() == coapCodes.NotFound {
		client.unpublishResources(ctx, []pbRA.ResourceId{pbRA.ResourceId{DeviceId: event.GetResourceId().GetDeviceId(), Href: event.GetResourceId().GetHref()}})
	}

	sendConfirmCtx, err := client.server.ServiceRequestContext(authCtx.UserID)
	if err != nil {
		return err
	}
	request := coapconv.MakeConfirmResourceUpdateRequest(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), event.GetCorrelationId(), authCtx.AuthorizationContext, client.remoteAddrString(), resp)
	_, err = client.server.raClient.ConfirmResourceUpdate(sendConfirmCtx, &request)
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) sendErrorConfirmResourceRetrieve(deviceID, href, userID, correlationID string, authCtx pbCQRS.AuthorizationContext, code codes.Code, errToSend error) {
	ctx, err := client.server.ServiceRequestContext(userID)
	if err != nil {
		log.Errorf("cannot send error via confirm resource retrieve: %v", err)
		return
	}
	resp := pool.AcquireMessage(ctx)
	defer pool.ReleaseMessage(resp)
	resp.SetContentFormat(message.TextPlain)
	resp.SetBody(bytes.NewReader([]byte(errToSend.Error())))
	resp.SetCode(code)
	request := coapconv.MakeConfirmResourceRetrieveRequest(deviceID, href, correlationID, authCtx, client.remoteAddrString(), resp)
	_, err = client.server.raClient.ConfirmResourceRetrieve(ctx, &request)
	if err != nil {
		log.Errorf("cannot send error confirm resource retrieve: %v", err)
	}
}

func (client *Client) retrieveResource(ctx context.Context, event *pb.Event_ResourceRetrievePending) error {
	authCtx := client.loadAuthorizationContext()
	if isExpired(authCtx.Expire) {
		err := fmt.Errorf("cannot retrieve resource /%v%v: token is expired", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref())
		client.sendErrorConfirmResourceUpdate(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.UserID, event.GetCorrelationId(), authCtx.AuthorizationContext, codes.Forbidden, err)
		client.Close()
		return err
	}

	if event.GetResourceId().GetHref() == cloud.StatusHref {
		authCtx := client.loadAuthorizationContext()
		msg := pool.AcquireMessage(ctx)
		msg.SetCode(coapCodes.Content)
		msg.SetSequence(client.coapConn.Sequence())
		defer pool.ReleaseMessage(msg)

		sendConfirmCtx, err := client.server.ServiceRequestContext(authCtx.UserID)
		if err != nil {
			return err
		}
		request := coapconv.MakeConfirmResourceRetrieveRequest(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), event.GetCorrelationId(), authCtx.AuthorizationContext, client.remoteAddrString(), msg)
		_, err = client.server.raClient.ConfirmResourceRetrieve(sendConfirmCtx, &request)
		if err != nil {
			return err
		}
		return nil
	}

	coapCtx, cancel := context.WithTimeout(ctx, client.server.RequestTimeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceRetrieveRequest(coapCtx, event)
	if err != nil {
		client.sendErrorConfirmResourceUpdate(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.UserID, event.GetCorrelationId(), authCtx.AuthorizationContext, codes.BadRequest, err)
		return err
	}
	defer pool.ReleaseMessage(req)

	decodeMsgToDebug(client, req, "RESOURCE-RETRIEVE-REQUEST")

	resp, err := client.coapConn.Do(req)
	if err != nil {
		client.sendErrorConfirmResourceUpdate(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.UserID, event.GetCorrelationId(), authCtx.AuthorizationContext, codes.ServiceUnavailable, err)
		return err
	}
	defer pool.ReleaseMessage(resp)

	decodeMsgToDebug(client, resp, "RESOURCE-RETRIEVE-RESPONSE")

	if resp.Code() == coapCodes.NotFound {
		client.unpublishResources(ctx, []pbRA.ResourceId{pbRA.ResourceId{
			DeviceId: event.GetResourceId().GetDeviceId(),
			Href:     event.GetResourceId().GetHref(),
		}})
	}

	sendConfirmCtx, err := client.server.ServiceRequestContext(authCtx.UserID)
	if err != nil {
		return err
	}
	request := coapconv.MakeConfirmResourceRetrieveRequest(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), event.GetCorrelationId(), authCtx.AuthorizationContext, client.remoteAddrString(), resp)
	_, err = client.server.raClient.ConfirmResourceRetrieve(sendConfirmCtx, &request)
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) sendErrorConfirmResourceDelete(deviceID, href, userID, correlationID string, authCtx pbCQRS.AuthorizationContext, code codes.Code, errToSend error) {
	ctx, err := client.server.ServiceRequestContext(userID)
	if err != nil {
		log.Errorf("cannot send error via confirm resource delete: %v", err)
		return
	}

	resp := pool.AcquireMessage(ctx)
	defer pool.ReleaseMessage(resp)
	resp.SetContentFormat(message.TextPlain)
	resp.SetBody(bytes.NewReader([]byte(errToSend.Error())))
	resp.SetCode(code)
	request := coapconv.MakeConfirmResourceDeleteRequest(deviceID, href, correlationID, authCtx, client.remoteAddrString(), resp)
	_, err = client.server.raClient.ConfirmResourceDelete(ctx, &request)
	if err != nil {
		log.Errorf("cannot send error via confirm resource delete: %v", err)
	}
}

func (client *Client) deleteResource(ctx context.Context, event *pb.Event_ResourceDeletePending) error {
	authCtx := client.loadAuthorizationContext()
	if isExpired(authCtx.Expire) {
		err := fmt.Errorf("cannot delete resource /%v%v: token is expired", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref())
		client.sendErrorConfirmResourceDelete(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.UserID, event.GetCorrelationId(), authCtx.AuthorizationContext, codes.Forbidden, err)
		client.Close()
		return err
	}

	if event.GetResourceId().GetHref() == cloud.StatusHref {
		authCtx := client.loadAuthorizationContext()
		msg := pool.AcquireMessage(ctx)
		msg.SetCode(coapCodes.Forbidden)
		msg.SetSequence(client.coapConn.Sequence())
		defer pool.ReleaseMessage(msg)

		sendConfirmCtx, err := client.server.ServiceRequestContext(authCtx.UserID)
		if err != nil {
			return err
		}
		request := coapconv.MakeConfirmResourceDeleteRequest(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), event.GetCorrelationId(), authCtx.AuthorizationContext, client.remoteAddrString(), msg)
		_, err = client.server.raClient.ConfirmResourceDelete(sendConfirmCtx, &request)
		if err != nil {
			return err
		}
		return nil
	}

	coapCtx, cancel := context.WithTimeout(ctx, client.server.RequestTimeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceDeleteRequest(coapCtx, event)
	if err != nil {
		client.sendErrorConfirmResourceDelete(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.UserID, event.GetCorrelationId(), authCtx.AuthorizationContext, codes.BadRequest, err)
		return err
	}
	defer pool.ReleaseMessage(req)

	decodeMsgToDebug(client, req, "RESOURCE-DELETE-REQUEST")

	resp, err := client.coapConn.Do(req)
	if err != nil {
		client.sendErrorConfirmResourceDelete(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.UserID, event.GetCorrelationId(), authCtx.AuthorizationContext, codes.ServiceUnavailable, err)
		return err
	}
	defer pool.ReleaseMessage(resp)

	decodeMsgToDebug(client, resp, "RESOURCE-DELETE-RESPONSE")

	if resp.Code() == coapCodes.NotFound {
		client.unpublishResources(ctx, []pbRA.ResourceId{pbRA.ResourceId{
			DeviceId: event.GetResourceId().GetDeviceId(),
			Href:     event.GetResourceId().GetHref(),
		}})
	}

	sendConfirmCtx, err := client.server.ServiceRequestContext(authCtx.UserID)
	if err != nil {
		return err
	}
	request := coapconv.MakeConfirmResourceDeleteRequest(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), event.GetCorrelationId(), authCtx.AuthorizationContext, client.remoteAddrString(), resp)
	_, err = client.server.raClient.ConfirmResourceDelete(sendConfirmCtx, &request)
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) publishResource(ctx context.Context, link schema.ResourceLink, ttl int32, connectionID string, sequence uint64, authCtx pbCQRS.AuthorizationContext) (schema.ResourceLink, error) {
	if link.DeviceID == "" {
		return link, fmt.Errorf("cannot send command publish resource: invalid DeviceId")
	}
	link.Href = fixHref(link.Href)

	if link.Href == "" || link.Href == "/" {
		return link, fmt.Errorf("cannot send command publish resource: invalid Href")
	}
	resourceID := resource2UUID(link.DeviceID, link.Href)

	raLink := pbGRPC.SchemaResourceLinkToProto(link).ToRAProto()
	raLink.Id = resourceID

	request := pbRA.PublishResourceRequest{
		AuthorizationContext: &authCtx,
		ResourceId: &pbRA.ResourceId{
			DeviceId: link.GetDeviceID(),
			Href:     link.Href,
		},
		Resource:   &raLink,
		TimeToLive: ttl,
		CommandMetadata: &pbCQRS.CommandMetadata{
			Sequence:     sequence,
			ConnectionId: connectionID,
		},
	}

	_, err := client.server.raClient.PublishResource(ctx, &request)
	if err != nil {
		return link, fmt.Errorf("cannot process command publish resource: %w", err)
	}

	link.InstanceID = getInstanceID(raLink.GetHref())
	link.ID = resourceID
	return link, nil
}

func (client *Client) unpublishResource(ctx context.Context, deviceID, href string, rscsUnpublished map[string]bool) map[string]bool {
	authCtx := client.loadAuthorizationContext()
	token, err := client.server.oauthMgr.GetToken(ctx)
	if err != nil {
		log.Errorf("ResourceId: %v: cannot unpublish resource /%v%v: %v", deviceID, href, err)
		rscsUnpublished[href] = false
		return rscsUnpublished
	}
	ctx = kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(ctx, token.AccessToken), authCtx.UserID)
	_, err = client.server.raClient.UnpublishResource(ctx, &pbRA.UnpublishResourceRequest{
		AuthorizationContext: &pbCQRS.AuthorizationContext{
			DeviceId: authCtx.DeviceId,
		},
		ResourceId: &pbRA.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CommandMetadata: &pbCQRS.CommandMetadata{
			ConnectionId: client.remoteAddrString(),
			Sequence:     client.coapConn.Sequence(),
		},
	})
	if err != nil {
		// unpublish resource is not critical -> resource is still accessible,
		// but next update of device resource will returns 'not found; and it triggers again unpublish.
		log.Errorf("ResourceId: %v: cannot unpublish resource /%v%v: %v", deviceID, href, err)
		rscsUnpublished[href] = false
		return rscsUnpublished
	}

	rscsUnpublished[href] = true
	return rscsUnpublished
}

func (client *Client) unpublishResources(ctx context.Context, resourceIDs []pbRA.ResourceId) {
	rscsUnpublished := make(map[string]bool, 32)

	for idx := range resourceIDs {
		rscsUnpublished = client.unpublishResource(ctx, resourceIDs[idx].DeviceId, resourceIDs[idx].Href, rscsUnpublished)
	}

	client.unobserveResources(ctx, resourceIDs, rscsUnpublished)
}
