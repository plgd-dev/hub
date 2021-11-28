package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/device/schema/resources"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/coap-gateway/coapconv"
	grpcClient "github.com/plgd-dev/hub/grpc-gateway/client"
	idEvents "github.com/plgd-dev/hub/identity-store/events"
	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	pkgJwt "github.com/plgd-dev/hub/pkg/security/jwt"
	"github.com/plgd-dev/hub/pkg/sync/task/future"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	kitSync "github.com/plgd-dev/kit/v2/sync"
)

type authorizationContext struct {
	DeviceID    string
	AccessToken string
	UserID      string
	Expire      time.Time
}

func (a *authorizationContext) GetUserID() string {
	if a == nil {
		return ""
	}
	return a.UserID
}

func (a *authorizationContext) GetDeviceID() string {
	if a != nil {
		return a.DeviceID
	}
	return ""
}

func (a *authorizationContext) GetAccessToken() string {
	if a != nil {
		return a.AccessToken
	}
	return ""
}

func (a *authorizationContext) IsValid() error {
	if a == nil {
		return fmt.Errorf("invalid authorization context")
	}
	if a.AccessToken == "" {
		return fmt.Errorf("invalid access token")
	}
	if !a.Expire.IsZero() && time.Now().UnixNano() > a.Expire.UnixNano() {
		return fmt.Errorf("token is expired")
	}
	return nil
}

func (a *authorizationContext) ToContext(ctx context.Context) context.Context {
	return kitNetGrpc.CtxWithToken(ctx, a.GetAccessToken())
}

//Client a setup of connection
type Client struct {
	server      *Service
	coapConn    *tcp.ClientConn
	tlsDeviceID string

	resourceSubscriptions *kitSync.Map // [token]

	exchangeCache *exchangeCache
	refreshCache  *refreshCache

	mutex                   sync.Mutex
	authCtx                 *authorizationContext
	deviceSubscriber        *grpcClient.DeviceSubscriber
	deviceObserver          *future.Future
	closeEventSubscriptions func()
}

//newClient create and initialize client
func newClient(server *Service, coapConn *tcp.ClientConn, tlsDeviceID string) *Client {
	return &Client{
		server:                server,
		coapConn:              coapConn,
		tlsDeviceID:           tlsDeviceID,
		resourceSubscriptions: kitSync.NewMap(),
		exchangeCache:         NewExchangeCache(),
		refreshCache:          NewRefreshCache(),
	}
}

func (client *Client) remoteAddrString() string {
	return client.coapConn.RemoteAddr().String()
}

func (client *Client) Context() context.Context {
	return client.coapConn.Context()
}

func (client *Client) cancelResourceSubscription(token string) (bool, error) {
	s, ok := client.resourceSubscriptions.PullOut(token)
	if !ok {
		return false, nil
	}
	sub := s.(*resourceSubscription)

	err := sub.Close()
	if err != nil {
		return false, err
	}
	return true, nil
}

// Callback executed when the Get response is received in the deviceObserver.
//
// This function is executed in the coap connection-goroutine, any operation on the connection (read, write, ...)
// will cause a deadlock . To avoid this problem the operation must be executed inside the taskQueue.
//
// The received notification is released by this function at the correct moment and must not be released
// by the caller.
func (client *Client) onGetResourceContent(ctx context.Context, deviceID, href string, notification *pool.Message) error {
	cannotGetResourceContentError := func(deviceID, href string, err error) error {
		return fmt.Errorf("cannot get resource /%v%v content: %w", deviceID, href, err)
	}
	notification.Hijack()
	err := client.server.taskQueue.Submit(func() {
		defer pool.ReleaseMessage(notification)
		err2 := client.notifyContentChanged(deviceID, href, false, notification)
		if err2 != nil {
			// cloud is unsynchronized against device. To recover cloud state, client need to reconnect to cloud.
			log.Error(cannotGetResourceContentError(deviceID, href, err2))
			if err3 := client.Close(); err3 != nil {
				log.Errorf("failed to close client connection on get resource /%v%v: %w", deviceID, href, err3)
			}
		}
		if notification.Code() == codes.NotFound {
			client.unpublishResourceLinks(client.getUserAuthorizedContext(ctx), []string{href})
		}
	})
	if err != nil {
		defer pool.ReleaseMessage(notification)
		return cannotGetResourceContentError(deviceID, href, err)
	}
	return nil
}

// Callback executed when the Observe notification is received in the deviceObserver.
//
// This function is executed in the coap connection-goroutine, any operation on the connection (read, write, ...)
// will cause a deadlock . To avoid this problem the operation must be executed inside the taskQueue.
//
// The received notification is released by this function at the correct moment and must not be released
// by the caller.
func (client *Client) onObserveResource(ctx context.Context, deviceID, href string, batch bool, notification *pool.Message) error {
	cannotObserResourceError := func(err error) error {
		return fmt.Errorf("cannot handle resource observation: %w", err)
	}
	notification.Hijack()
	err := client.server.taskQueue.Submit(func() {
		defer pool.ReleaseMessage(notification)
		err2 := client.notifyContentChanged(deviceID, href, batch, notification)
		if err2 != nil {
			// cloud is unsynchronized against device. To recover cloud state, client need to reconnect to cloud.
			log.Error(cannotObserResourceError(err2))
			if err3 := client.Close(); err3 != nil {
				log.Errorf("failed to close client connection on resource /%v%v observation: %w", deviceID, href, err3)
			}
		}
		if notification.Code() == codes.NotFound {
			client.unpublishResourceLinks(client.getUserAuthorizedContext(notification.Context()), []string{href})
		}
	})
	if err != nil {
		defer pool.ReleaseMessage(notification)
		return cannotObserResourceError(err)
	}
	return nil
}

// Close closes coap connection
func (client *Client) Close() error {
	err := client.coapConn.Close()
	if err != nil {
		return fmt.Errorf("cannot close client: %w", err)
	}
	return nil
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
			log.Errorf("cannot cancel resource subscription: %w", err)
		} else if wantWait {
			wait()
		}
	}
}

func (client *Client) CleanUp(resetAuthContext bool) *authorizationContext {
	authCtx, _ := client.GetAuthorizationContext()
	log.Debugf("cleanUp client %v for device %v", client.coapConn.RemoteAddr(), authCtx.GetDeviceID())

	client.server.devicesStatusUpdater.Remove(client)
	if err := client.closeDeviceObserver(client.Context()); err != nil {
		log.Errorf("cleanUp error: failed to close observer for device %v: %w", authCtx.GetDeviceID(), err)
	}
	client.cancelResourceSubscriptions(false)
	if err := client.closeDeviceSubscriber(); err != nil {
		log.Errorf("cleanUp error: failed to close device %v connection: %w", authCtx.GetDeviceID(), err)
	}
	client.unsubscribeFromDeviceEvents()

	if resetAuthContext {
		return client.SetAuthorizationContext(nil)
	}
	// we cannot reset authorizationContext need token (eg signOff)
	return authCtx
}

// OnClose action when coap connection was closed.
func (client *Client) OnClose() {
	authCtx, _ := client.GetAuthorizationContext()
	log.Debugf("close client %v for device %v", client.coapConn.RemoteAddr(), authCtx.GetDeviceID())
	oldAuthCtx := client.CleanUp(false)

	if oldAuthCtx.GetDeviceID() != "" {
		client.server.expirationClientCache.Set(oldAuthCtx.GetDeviceID(), nil, time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), client.server.config.APIs.COAP.KeepAlive.Timeout)
		defer cancel()
		_, err := client.server.raClient.UpdateDeviceMetadata(kitNetGrpc.CtxWithToken(ctx, oldAuthCtx.GetAccessToken()), &commands.UpdateDeviceMetadataRequest{
			DeviceId: authCtx.GetDeviceID(),
			Update: &commands.UpdateDeviceMetadataRequest_Status{
				Status: &commands.ConnectionStatus{
					Value: commands.ConnectionStatus_OFFLINE,
				},
			},
			CommandMetadata: &commands.CommandMetadata{
				Sequence:     client.coapConn.Sequence(),
				ConnectionId: client.remoteAddrString(),
			},
		})
		if err != nil {
			// Device will be still reported as online and it can fix his state by next calls online, offline commands.
			log.Errorf("DeviceId %v: cannot handle sign out: cannot update cloud device status: %w", oldAuthCtx.GetDeviceID(), err)
		}
	}
}

func (client *Client) SetAuthorizationContext(authCtx *authorizationContext) (oldDeviceID *authorizationContext) {
	log.Debugf("Authorization context replaced for client %v, device %v, user %v", client.coapConn.RemoteAddr(), authCtx.GetDeviceID(), authCtx.GetUserID())
	client.mutex.Lock()
	defer client.mutex.Unlock()
	oldAuthContext := client.authCtx
	client.authCtx = authCtx
	return oldAuthContext
}

func (client *Client) GetAuthorizationContext() (*authorizationContext, error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	return client.authCtx, client.authCtx.IsValid()
}

func (client *Client) notifyContentChanged(deviceID, href string, batch bool, notification *pool.Message) error {
	notifyError := func(deviceID, href string, err error) error {
		return fmt.Errorf("cannot notify resource /%v%v content changed: %w", deviceID, href, err)
	}
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		return notifyError(deviceID, href, err)
	}
	decodeMsgToDebug(client, notification, "RECEIVED-NOTIFICATION")

	var requests []*commands.NotifyResourceChangedRequest
	if batch && href == resources.ResourceURI {
		requests, err = coapconv.NewNotifyResourceChangedRequests(deviceID, client.remoteAddrString(), notification)
		if err != nil {
			return notifyError(deviceID, href, err)
		}
	} else {
		requests = []*commands.NotifyResourceChangedRequest{coapconv.NewNotifyResourceChangedRequest(commands.NewResourceID(deviceID, href), client.remoteAddrString(), notification)}
	}

	ctx := kitNetGrpc.CtxWithToken(client.Context(), authCtx.GetAccessToken())
	for _, request := range requests {
		_, err = client.server.raClient.NotifyResourceChanged(ctx, request)
		if err != nil {
			return notifyError(request.GetResourceId().GetDeviceId(), request.GetResourceId().GetHref(), err)
		}
	}
	return nil
}

func (client *Client) sendErrorConfirmResourceUpdate(ctx context.Context, deviceID, href, userID, correlationID string, code codes.Code, errToSend error) {
	resp := pool.AcquireMessage(ctx)
	defer pool.ReleaseMessage(resp)
	resp.SetContentFormat(message.TextPlain)
	resp.SetBody(bytes.NewReader([]byte(errToSend.Error())))
	resp.SetCode(code)
	request := coapconv.NewConfirmResourceUpdateRequest(commands.NewResourceID(deviceID, href), correlationID, client.remoteAddrString(), resp)
	_, err := client.server.raClient.ConfirmResourceUpdate(ctx, request)
	if err != nil {
		log.Errorf("cannot send error via confirm resource update: %w", err)
	}
}

func (client *Client) UpdateResource(ctx context.Context, event *events.ResourceUpdatePending) error {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		err := fmt.Errorf("cannot update resource /%v%v: %w", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), err)
		if err2 := client.Close(); err2 != nil {
			log.Errorf("failed to close client connection on update resource /%v%v: %w", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), err2)
		}
		return err
	}
	sendConfirmCtx := authCtx.ToContext(ctx)

	if event.GetResourceId().GetHref() == commands.StatusHref {
		msg := pool.AcquireMessage(ctx)
		msg.SetCode(codes.MethodNotAllowed)
		msg.SetSequence(client.coapConn.Sequence())
		defer pool.ReleaseMessage(msg)
		request := coapconv.NewConfirmResourceUpdateRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), client.remoteAddrString(), msg)
		_, err = client.server.raClient.ConfirmResourceUpdate(sendConfirmCtx, request)
		if err != nil {
			return err
		}
		return nil
	}

	coapCtx, cancel := context.WithTimeout(ctx, client.server.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceUpdateRequest(coapCtx, event)
	if err != nil {
		client.sendErrorConfirmResourceUpdate(sendConfirmCtx, event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.GetUserID(), event.GetAuditContext().GetCorrelationId(), codes.BadRequest, err)
		return err
	}
	defer pool.ReleaseMessage(req)

	decodeMsgToDebug(client, req, "RESOURCE-UPDATE-REQUEST")

	resp, err := client.coapConn.Do(req)
	if err != nil {
		client.sendErrorConfirmResourceUpdate(sendConfirmCtx, event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.GetUserID(), event.GetAuditContext().GetCorrelationId(), codes.ServiceUnavailable, err)
		return err
	}
	defer pool.ReleaseMessage(resp)

	decodeMsgToDebug(client, resp, "RESOURCE-UPDATE-RESPONSE")

	if resp.Code() == codes.NotFound {
		client.unpublishResourceLinks(client.getUserAuthorizedContext(ctx), []string{event.GetResourceId().GetHref()})
	}

	request := coapconv.NewConfirmResourceUpdateRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), client.remoteAddrString(), resp)
	_, err = client.server.raClient.ConfirmResourceUpdate(sendConfirmCtx, request)
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) sendErrorConfirmResourceRetrieve(ctx context.Context, deviceID, href, userID, correlationID string, code codes.Code, errToSend error) {
	resp := pool.AcquireMessage(ctx)
	defer pool.ReleaseMessage(resp)
	resp.SetContentFormat(message.TextPlain)
	resp.SetBody(bytes.NewReader([]byte(errToSend.Error())))
	resp.SetCode(code)
	request := coapconv.NewConfirmResourceRetrieveRequest(commands.NewResourceID(deviceID, href), correlationID, client.remoteAddrString(), resp)
	_, err := client.server.raClient.ConfirmResourceRetrieve(ctx, request)
	if err != nil {
		log.Errorf("cannot send error confirm resource retrieve: %w", err)
	}
}

func (client *Client) RetrieveResource(ctx context.Context, event *events.ResourceRetrievePending) error {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		err := fmt.Errorf("cannot retrieve resource /%v%v: %w", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), err)
		if err2 := client.Close(); err2 != nil {
			log.Errorf("failed to close client connection on retrieve resource /%v%v: %w", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), err2)
		}
		return err
	}
	sendConfirmCtx := authCtx.ToContext(ctx)

	if event.GetResourceId().GetHref() == commands.StatusHref {
		msg := pool.AcquireMessage(ctx)
		msg.SetCode(codes.Content)
		msg.SetSequence(client.coapConn.Sequence())
		defer pool.ReleaseMessage(msg)

		request := coapconv.NewConfirmResourceRetrieveRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), client.remoteAddrString(), msg)
		_, err = client.server.raClient.ConfirmResourceRetrieve(sendConfirmCtx, request)
		if err != nil {
			return err
		}
		return nil
	}

	coapCtx, cancel := context.WithTimeout(ctx, client.server.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceRetrieveRequest(coapCtx, event)
	if err != nil {
		client.sendErrorConfirmResourceRetrieve(sendConfirmCtx, event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.GetUserID(), event.GetAuditContext().GetCorrelationId(), codes.BadRequest, err)
		return err
	}
	defer pool.ReleaseMessage(req)

	decodeMsgToDebug(client, req, "RESOURCE-RETRIEVE-REQUEST")

	resp, err := client.coapConn.Do(req)
	if err != nil {
		client.sendErrorConfirmResourceRetrieve(sendConfirmCtx, event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.GetUserID(), event.GetAuditContext().GetCorrelationId(), codes.ServiceUnavailable, err)
		return err
	}
	defer pool.ReleaseMessage(resp)

	decodeMsgToDebug(client, resp, "RESOURCE-RETRIEVE-RESPONSE")

	if resp.Code() == codes.NotFound {
		client.unpublishResourceLinks(client.getUserAuthorizedContext(ctx), []string{event.GetResourceId().GetHref()})
	}

	request := coapconv.NewConfirmResourceRetrieveRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), client.remoteAddrString(), resp)
	_, err = client.server.raClient.ConfirmResourceRetrieve(sendConfirmCtx, request)
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) sendErrorConfirmResourceDelete(ctx context.Context, deviceID, href, userID, correlationID string, code codes.Code, errToSend error) {
	resp := pool.AcquireMessage(ctx)
	defer pool.ReleaseMessage(resp)
	resp.SetContentFormat(message.TextPlain)
	resp.SetBody(bytes.NewReader([]byte(errToSend.Error())))
	resp.SetCode(code)
	request := coapconv.NewConfirmResourceDeleteRequest(commands.NewResourceID(deviceID, href), correlationID, client.remoteAddrString(), resp)
	_, err := client.server.raClient.ConfirmResourceDelete(ctx, request)
	if err != nil {
		log.Errorf("cannot send error via confirm resource delete: %w", err)
	}
}

func (client *Client) DeleteResource(ctx context.Context, event *events.ResourceDeletePending) error {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		err := fmt.Errorf("cannot delete resource /%v%v: %w", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), err)
		if err2 := client.Close(); err2 != nil {
			log.Errorf("failed to close client connection on delete resource /%v%v: %w", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), err2)
		}
		return err
	}
	sendConfirmCtx := authCtx.ToContext(ctx)

	if event.GetResourceId().GetHref() == commands.StatusHref {
		msg := pool.AcquireMessage(ctx)
		msg.SetCode(codes.Forbidden)
		msg.SetSequence(client.coapConn.Sequence())
		defer pool.ReleaseMessage(msg)

		request := coapconv.NewConfirmResourceDeleteRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), client.remoteAddrString(), msg)
		_, err = client.server.raClient.ConfirmResourceDelete(sendConfirmCtx, request)
		if err != nil {
			return err
		}
		return nil
	}

	coapCtx, cancel := context.WithTimeout(ctx, client.server.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceDeleteRequest(coapCtx, event)
	if err != nil {
		client.sendErrorConfirmResourceDelete(sendConfirmCtx, event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.GetUserID(), event.GetAuditContext().GetCorrelationId(), codes.BadRequest, err)
		return err
	}
	defer pool.ReleaseMessage(req)

	decodeMsgToDebug(client, req, "RESOURCE-DELETE-REQUEST")

	resp, err := client.coapConn.Do(req)
	if err != nil {
		client.sendErrorConfirmResourceDelete(sendConfirmCtx, event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), authCtx.GetUserID(), event.GetAuditContext().GetCorrelationId(), codes.ServiceUnavailable, err)
		return err
	}
	defer pool.ReleaseMessage(resp)

	decodeMsgToDebug(client, resp, "RESOURCE-DELETE-RESPONSE")

	if resp.Code() == codes.NotFound {
		client.unpublishResourceLinks(client.getUserAuthorizedContext(ctx), []string{event.GetResourceId().GetHref()})
	}

	request := coapconv.NewConfirmResourceDeleteRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), client.remoteAddrString(), resp)
	_, err = client.server.raClient.ConfirmResourceDelete(sendConfirmCtx, request)
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) getUserAuthorizedContext(ctx context.Context) context.Context {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		log.Errorf("unable to load authorization context: %w", err)
		return nil
	}

	return authCtx.ToContext(ctx)
}

func (client *Client) unpublishResourceLinks(ctx context.Context, hrefs []string) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		log.Errorf("unable to load authorization context during resource links publish for device: %w", err)
		return
	}

	logUnpublishError := func(err error) {
		log.Errorf("error occurred during resource links unpublish for device %v: %w", authCtx.GetDeviceID(), err)
	}
	resp, err := client.server.raClient.UnpublishResourceLinks(ctx, &commands.UnpublishResourceLinksRequest{
		Hrefs:    hrefs,
		DeviceId: authCtx.GetDeviceID(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: client.remoteAddrString(),
			Sequence:     client.coapConn.Sequence(),
		},
	})
	if err != nil {
		// unpublish resource is not critical -> resource can be still accessible
		// next resource update will return 'not found' what triggers a publish again
		logUnpublishError(err)
		return
	}

	observer, err := client.getDeviceObserver(ctx)
	if err != nil {
		logUnpublishError(err)
		return
	}
	observer.RemovePublishedResources(ctx, resp.UnpublishedHrefs)
}

func (client *Client) sendErrorConfirmResourceCreate(ctx context.Context, resourceID *commands.ResourceId, userID, correlationID string, code codes.Code, errToSend error) {
	resp := pool.AcquireMessage(ctx)
	defer pool.ReleaseMessage(resp)
	resp.SetContentFormat(message.TextPlain)
	resp.SetBody(bytes.NewReader([]byte(errToSend.Error())))
	resp.SetCode(code)
	request := coapconv.NewConfirmResourceCreateRequest(resourceID, correlationID, client.remoteAddrString(), resp)
	_, err := client.server.raClient.ConfirmResourceCreate(ctx, request)
	if err != nil {
		log.Errorf("cannot send error via confirm resource create: %w", err)
	}
}

func (client *Client) CreateResource(ctx context.Context, event *events.ResourceCreatePending) error {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		err := fmt.Errorf("cannot create resource /%v%v: %w", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), err)
		if err2 := client.Close(); err2 != nil {
			log.Errorf("failed to close client connection on create resource /%v%v: %w", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), err2)
		}
		return err
	}
	sendConfirmCtx := authCtx.ToContext(ctx)
	if event.GetResourceId().GetHref() == commands.StatusHref {
		msg := pool.AcquireMessage(ctx)
		msg.SetCode(codes.Forbidden)
		msg.SetSequence(client.coapConn.Sequence())
		defer pool.ReleaseMessage(msg)
		request := coapconv.NewConfirmResourceCreateRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), client.remoteAddrString(), msg)
		_, err = client.server.raClient.ConfirmResourceCreate(sendConfirmCtx, request)
		if err != nil {
			return err
		}
		return nil
	}

	coapCtx, cancel := context.WithTimeout(ctx, client.server.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceCreateRequest(coapCtx, event)
	if err != nil {
		client.sendErrorConfirmResourceCreate(sendConfirmCtx, event.GetResourceId(), authCtx.GetUserID(), event.GetAuditContext().GetCorrelationId(), codes.BadRequest, err)
		return err
	}
	defer pool.ReleaseMessage(req)

	decodeMsgToDebug(client, req, "RESOURCE-CREATE-REQUEST")

	resp, err := client.coapConn.Do(req)
	if err != nil {
		client.sendErrorConfirmResourceCreate(sendConfirmCtx, event.GetResourceId(), authCtx.GetUserID(), event.GetAuditContext().GetCorrelationId(), codes.ServiceUnavailable, err)
		return err
	}
	defer pool.ReleaseMessage(resp)

	decodeMsgToDebug(client, resp, "RESOURCE-CREATE-RESPONSE")

	if resp.Code() == codes.NotFound {
		client.unpublishResourceLinks(client.getUserAuthorizedContext(ctx), []string{event.GetResourceId().GetHref()})
	}

	request := coapconv.NewConfirmResourceCreateRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), client.remoteAddrString(), resp)
	_, err = client.server.raClient.ConfirmResourceCreate(sendConfirmCtx, request)
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) OnDeviceSubscriberReconnectError(err error) {
	auth, _ := client.GetAuthorizationContext()
	deviceID := auth.GetDeviceID()
	log.Errorf("cannot reconnect device %v subscriber to resource directory or eventbus - closing the device connection: %w", deviceID, err)
	if err := client.Close(); err != nil {
		log.Errorf("failed to close device %v connection : %w", deviceID, err)
	}
}

func (client *Client) GetContext() context.Context {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		return client.Context()
	}
	return authCtx.ToContext(client.Context())
}

func (client *Client) UpdateDeviceMetadata(ctx context.Context, event *events.DeviceMetadataUpdatePending) error {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		err := fmt.Errorf("cannot update device('%v') metadata: %w", event.GetDeviceId(), err)
		if err2 := client.Close(); err2 != nil {
			log.Errorf("failed to close client connection on update device('%v') metadata: %w", event.GetDeviceId(), err2)
		}
		return err
	}
	if event.GetShadowSynchronization() == commands.ShadowSynchronization_UNSET {
		return nil
	}
	sendConfirmCtx := authCtx.ToContext(ctx)

	previous, errObs := client.replaceDeviceObserverWithDeviceShadow(sendConfirmCtx, event.GetShadowSynchronization())
	if errObs != nil {
		log.Errorf("update device('%v') metadata error: %w", event.GetDeviceId(), errObs)
	}
	_, err = client.server.raClient.ConfirmDeviceMetadataUpdate(sendConfirmCtx, &commands.ConfirmDeviceMetadataUpdateRequest{
		DeviceId:      event.GetDeviceId(),
		CorrelationId: event.GetAuditContext().GetCorrelationId(),
		Confirm: &commands.ConfirmDeviceMetadataUpdateRequest_ShadowSynchronization{
			ShadowSynchronization: event.GetShadowSynchronization(),
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: client.remoteAddrString(),
			Sequence:     client.coapConn.Sequence(),
		},
		Status: commands.Status_OK,
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		_, errObs := client.replaceDeviceObserverWithDeviceShadow(sendConfirmCtx, previous)
		if errObs != nil {
			log.Errorf("update device('%v') metadata error: %w", event.GetDeviceId(), errObs)
		}
	}
	return err
}

func (c *Client) ValidateToken(ctx context.Context, token string) (pkgJwt.Claims, error) {
	return c.server.ValidateToken(ctx, token)
}

func (c *Client) subscribeToDeviceEvents(owner string, onEvent func(e *idEvents.Event)) error {
	close, err := c.server.ownerCache.Subscribe(owner, onEvent)
	if err != nil {
		return err
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.closeEventSubscriptions = close
	return nil
}

func (c *Client) unsubscribeFromDeviceEvents() {
	close := func() {
		// default no-op
	}
	c.mutex.Lock()
	if c.closeEventSubscriptions != nil {
		close = c.closeEventSubscriptions
		c.closeEventSubscriptions = nil
	}
	c.mutex.Unlock()
	close()
}

func (c *Client) ResolveDeviceID(claim pkgJwt.Claims, paramDeviceID string) string {
	if c.server.config.APIs.COAP.Authorization.DeviceIDClaim != "" {
		return claim.DeviceID(c.server.config.APIs.COAP.Authorization.DeviceIDClaim)
	}
	if c.server.config.APIs.COAP.TLS.Enabled && c.server.config.APIs.COAP.TLS.Embedded.ClientCertificateRequired {
		return c.tlsDeviceID
	}
	return paramDeviceID
}
