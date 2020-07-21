package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	grpcClient "github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	pbGRPC "github.com/go-ocf/cloud/grpc-gateway/pb"
	cqrsRA "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/tcp"
	"github.com/go-ocf/go-coap/v2/tcp/message/pool"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	kitSync "github.com/go-ocf/kit/sync"
	"github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/schema/cloud"
)

type observedResource struct {
	res         *pbRA.Resource
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

func (client *Client) observeResource(ctx context.Context, res *pbRA.Resource, allowDuplicit bool) (err error) {
	log.Debugf("DeviceId: %v, ResourceId: %v: observe resource", res.DeviceId, res.Id)

	client.observedResourcesLock.Lock()
	defer client.observedResourcesLock.Unlock()
	if _, ok := client.observedResources[res.DeviceId]; !ok {
		client.observedResources[res.DeviceId] = make(map[int64]observedResource)
	}
	if _, ok := client.observedResources[res.DeviceId][res.InstanceId]; ok {
		if allowDuplicit {
			return nil
		}
		return fmt.Errorf("resource is already already published")
	}
	return client.addObservedResourceLocked(ctx, res)
}

func (client *Client) getResourceContent(ctx context.Context, obsRes *pbRA.Resource) {
	resp, err := client.coapConn.Get(ctx, obsRes.Href)
	if err != nil {
		log.Errorf("DeviceId: %v, ResourceId: %v: cannot get resource content: %v", obsRes.DeviceId, obsRes.Id, err)
		return
	}
	defer pool.ReleaseMessage(resp)
	err = client.notifyContentChanged(obsRes, resp)
	if err != nil {
		// cloud is unsynchronized against device. To recover cloud state, client need to reconnect to cloud.
		log.Errorf("DeviceId: %v, ResourceId: %v: cannot get resource content: %v", obsRes.DeviceId, obsRes.Id, err)
		client.Close()
	}
	if resp.Code() == coapCodes.NotFound {
		client.unpublishResources(ctx, []string{obsRes.GetId()})
	}
}

func (client *Client) addObservedResourceLocked(ctx context.Context, res *pbRA.Resource) error {
	var observation *tcp.Observation
	obs := isObservable(res)
	log.Debugf("DeviceId: %v, ResourceId: %v: Observable: %v: Client.addObservedResourceLocked", res.DeviceId, res.Href, obs)

	if res.Id == cqrsRA.MakeResourceId(res.DeviceId, cloud.StatusHref) {
		return nil
	}

	obsRes := res.Clone()
	if obs {
		obs, err := client.coapConn.Observe(ctx, res.Href, func(req *pool.Message) {
			err := client.notifyContentChanged(obsRes, req)
			if err != nil {
				// cloud is unsynchronized against device. To recover cloud state, client need to reconnect to cloud.
				log.Errorf("DeviceId: %v, ResourceId: %v: cannot get resource content %v%v: %v", obsRes.DeviceId, obsRes.Id, obsRes.DeviceId, obsRes.Href, err)
				client.Close()
			}
			if req.Code() == coapCodes.NotFound {
				client.unpublishResources(req.Context(), []string{obsRes.GetId()})
			}
		})
		if err != nil {
			log.Errorf("DeviceId: %v, ResourceId: %v: cannot observe resource %v%v: %v", obsRes.DeviceId, obsRes.Id, obsRes.DeviceId, obsRes.Href, err)
		} else {
			observation = obs
		}
	} else {
		go client.getResourceContent(ctx, obsRes)
	}
	client.observedResources[res.DeviceId][res.InstanceId] = observedResource{res: obsRes, observation: observation}
	return nil
}

func (client *Client) getObservedResources(deviceID string, instanceIDs []int64) []string {
	client.observedResourcesLock.Lock()
	defer client.observedResourcesLock.Unlock()

	getAllDeviceIDMatches := len(instanceIDs) == 0
	matches := make([]string, 0, 16)

	if deviceResourcesMap, ok := client.observedResources[deviceID]; ok {
		if getAllDeviceIDMatches {
			for _, value := range deviceResourcesMap {
				matches = append(matches, value.res.GetId())
			}
		} else {
			for _, instanceID := range instanceIDs {
				if resource, ok := deviceResourcesMap[instanceID]; ok {
					matches = append(matches, resource.res.GetId())
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

func (client *Client) unobserveResources(ctx context.Context, resourceIDs []string, rscsUnpublished map[string]bool) {
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

func (client *Client) unobserveAndRemoveResources(resourceIDs []string, rscsUnpublished map[string]bool) []*tcp.Observation {
	observartions := make([]*tcp.Observation, 0, 32)

	client.observedResourcesLock.Lock()
	defer client.observedResourcesLock.Unlock()

	for _, resourceID := range resourceIDs {
		if del, ok := rscsUnpublished[resourceID]; ok && del {
			log.Debugf("ResourceId: %v: delete resource", resourceID)
		} else {
			log.Debugf("ResourceId: %v: unobserve resource", resourceID)
		}
		var instanceID int64
		var deviceID string
		for devID, devs := range client.observedResources {
			for insID, r := range devs {
				if r.res.GetId() == resourceID {
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
		if rscsUnpublished[resourceID] {
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

func (client *Client) notifyContentChanged(res *pbRA.Resource, notification *pool.Message) error {
	authCtx := client.loadAuthorizationContext()
	if isExpired(authCtx.Expire) {
		return fmt.Errorf("cannot notify resource /%v%v content changed: token is expired", res.GetDeviceId(), res.GetHref())
	}

	decodeMsgToDebug(client, notification, "RECEIVED-NOTIFICATION")

	token, err := client.server.oauthMgr.GetToken(notification.Context())
	if err != nil {
		return fmt.Errorf("cannot notify resource /%v%v content changed: %w", res.GetDeviceId(), res.GetHref(), err)
	}
	ctx := kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(notification.Context(), token.AccessToken), authCtx.UserID)
	request := coapconv.MakeNotifyResourceChangedRequest(res.Id, authCtx.AuthorizationContext, client.remoteAddrString(), notification)
	_, err = client.server.raClient.NotifyResourceChanged(ctx, &request)
	if err != nil {
		return fmt.Errorf("cannot notify resource /%v%v content changed: %w", res.GetDeviceId(), res.GetHref(), err)
	}
	return nil
}

func (client *Client) updateResource(ctx context.Context, event *pb.Event_ResourceUpdatePending) error {
	resourceID := cqrsRA.MakeResourceId(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref())
	if event.GetResourceId().GetHref() == cloud.StatusHref {
		authCtx := client.loadAuthorizationContext()
		msg := pool.AcquireMessage(ctx)
		msg.SetCode(coapCodes.MethodNotAllowed)
		msg.SetSequence(client.coapConn.Sequence())
		defer pool.ReleaseMessage(msg)
		request := coapconv.MakeConfirmResourceUpdateRequest(resourceID, event.GetCorrelationId(), authCtx.AuthorizationContext, client.remoteAddrString(), msg)
		_, err := client.server.raClient.ConfirmResourceUpdate(ctx, &request)
		if err != nil {
			return err
		}
		return nil
	}
	authCtx := client.loadAuthorizationContext()
	if isExpired(authCtx.Expire) {
		client.Close()
		return fmt.Errorf("cannot update resource /%v%v: token is expired", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref())
	}

	ctx, cancel := context.WithTimeout(ctx, client.server.RequestTimeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceUpdateRequest(ctx, event)
	if err != nil {
		return err
	}
	defer pool.ReleaseMessage(req)

	resp, err := client.coapConn.Do(req)
	if err != nil {
		return err
	}
	defer pool.ReleaseMessage(resp)

	decodeMsgToDebug(client, resp, "RESOURCE-UPDATE")

	if resp.Code() == coapCodes.NotFound {
		client.unpublishResources(ctx, []string{resourceID})
	}

	request := coapconv.MakeConfirmResourceUpdateRequest(resourceID, event.GetCorrelationId(), authCtx.AuthorizationContext, client.remoteAddrString(), resp)
	_, err = client.server.raClient.ConfirmResourceUpdate(ctx, &request)
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) retrieveResource(ctx context.Context, event *pb.Event_ResourceRetrievePending) error {
	resourceID := cqrsRA.MakeResourceId(event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref())
	if event.GetResourceId().GetHref() == cloud.StatusHref {
		authCtx := client.loadAuthorizationContext()
		msg := pool.AcquireMessage(ctx)
		msg.SetCode(coapCodes.Content)
		msg.SetSequence(client.coapConn.Sequence())
		defer pool.ReleaseMessage(msg)

		request := coapconv.MakeConfirmResourceRetrieveRequest(resourceID, event.GetCorrelationId(), authCtx.AuthorizationContext, client.remoteAddrString(), msg)
		_, err := client.server.raClient.ConfirmResourceRetrieve(ctx, &request)
		if err != nil {
			return err
		}
		return nil
	}
	authCtx := client.loadAuthorizationContext()
	if isExpired(authCtx.Expire) {
		client.Close()
		return fmt.Errorf("cannot retrieve resource /%v%v: token is expired", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref())
	}

	ctx, cancel := context.WithTimeout(ctx, client.server.RequestTimeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceRetrieveRequest(ctx, event)
	if err != nil {
		return err
	}
	defer pool.ReleaseMessage(req)

	resp, err := client.coapConn.Do(req)
	if err != nil {
		return err
	}
	defer pool.ReleaseMessage(resp)

	decodeMsgToDebug(client, resp, "RESOURCE-RETRIEVE")

	if resp.Code() == coapCodes.NotFound {
		client.unpublishResources(ctx, []string{resourceID})
	}

	request := coapconv.MakeConfirmResourceRetrieveRequest(resourceID, event.GetCorrelationId(), authCtx.AuthorizationContext, client.remoteAddrString(), resp)
	_, err = client.server.raClient.ConfirmResourceRetrieve(ctx, &request)
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
		ResourceId:           resourceID,
		Resource:             &raLink,
		TimeToLive:           ttl,
		CommandMetadata: &pbCQRS.CommandMetadata{
			Sequence:     sequence,
			ConnectionId: connectionID,
		},
	}

	response, err := client.server.raClient.PublishResource(ctx, &request)
	if err != nil {
		return link, fmt.Errorf("cannot process command publish resource: %w", err)
	}

	link.InstanceID = response.InstanceId
	link.ID = resourceID
	return link, nil
}

func (client *Client) unpublishResource(ctx context.Context, resourceID string, rscsUnpublished map[string]bool) map[string]bool {
	authCtx := client.loadAuthorizationContext()
	token, err := client.server.oauthMgr.GetToken(ctx)
	if err != nil {
		log.Errorf("ResourceId: %v: cannot unpublish resource: %v", resourceID, err)
		rscsUnpublished[resourceID] = false
		return rscsUnpublished
	}
	ctx = kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(ctx, token.AccessToken), authCtx.UserID)
	_, err = client.server.raClient.UnpublishResource(ctx, &pbRA.UnpublishResourceRequest{
		AuthorizationContext: &pbCQRS.AuthorizationContext{
			DeviceId: authCtx.DeviceId,
		},
		ResourceId: resourceID,
		CommandMetadata: &pbCQRS.CommandMetadata{
			ConnectionId: client.remoteAddrString(),
			Sequence:     client.coapConn.Sequence(),
		},
	})
	if err != nil {
		// unpublish resource is not critical -> resource is still accessible,
		// but next update of device resource will returns 'not found; and it triggers again unpublish.
		log.Errorf("ResourceId: %v: cannot unpublish resource: %v", resourceID, err)
		rscsUnpublished[resourceID] = false
		return rscsUnpublished
	}

	rscsUnpublished[resourceID] = true
	return rscsUnpublished
}

func (client *Client) unpublishResources(ctx context.Context, resourceIDs []string) {
	rscsUnpublished := make(map[string]bool, 32)

	for _, resourceID := range resourceIDs {
		rscsUnpublished = client.unpublishResource(ctx, resourceID, rscsUnpublished)
	}

	client.unobserveResources(ctx, resourceIDs, rscsUnpublished)
}
