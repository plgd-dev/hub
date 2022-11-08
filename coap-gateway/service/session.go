package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	coapTcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/coap-gateway/resource"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/observation"
	grpcClient "github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	idEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/log"
	coapService "github.com/plgd-dev/hub/v2/pkg/net/coap/service"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry/otelcoap"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/future"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	kitSync "github.com/plgd-dev/kit/v2/sync"
	otelCodes "go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

type authorizationContext struct {
	DeviceID    string
	AccessToken string
	UserID      string
	Expire      time.Time
	JWTClaims   pkgJwt.Claims
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

func (a *authorizationContext) GetJWTClaims() pkgJwt.Claims {
	if a != nil {
		return a.JWTClaims
	}
	return make(pkgJwt.Claims)
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

// session a setup of connection
type session struct {
	server        *Service
	coapConn      mux.Conn
	tlsDeviceID   string
	tlsValidUntil time.Time

	resourceSubscriptions *kitSync.Map // [token]

	exchangeCache *exchangeCache
	refreshCache  *refreshCache

	mutex                   sync.Mutex
	authCtx                 *authorizationContext
	deviceSubscriber        *grpcClient.DeviceSubscriber
	deviceObserver          *future.Future
	closeEventSubscriptions func()
}

// newSession creates and initializes client
func newSession(server *Service, coapConn mux.Conn, tlsDeviceID string, tlsValidUntil time.Time) *session {
	return &session{
		server:                server,
		coapConn:              coapConn,
		tlsDeviceID:           tlsDeviceID,
		resourceSubscriptions: kitSync.NewMap(),
		exchangeCache:         NewExchangeCache(),
		refreshCache:          NewRefreshCache(),
		tlsValidUntil:         tlsValidUntil,
	}
}

func (c *session) deviceID() string {
	if c.tlsDeviceID != "" {
		return c.tlsDeviceID
	}
	a, err := c.GetAuthorizationContext()
	if err == nil {
		return a.GetDeviceID()
	}
	return ""
}

func (c *session) Protocol() coapService.Protocol {
	if _, ok := c.coapConn.(*coapTcpClient.Conn); ok {
		return coapService.TCP
	}
	return coapService.UDP
}

func (c *session) getSessionExpiration(validJWTUntil time.Time) time.Time {
	if c.server.config.APIs.COAP.TLS.IsEnabled() &&
		c.server.config.APIs.COAP.TLS.DisconnectOnExpiredCertificate &&
		(validJWTUntil.IsZero() || validJWTUntil.After(c.tlsValidUntil)) {
		return c.tlsValidUntil
	}
	return validJWTUntil
}

func (c *session) WriteMessage(msg *pool.Message) {
	if err := c.coapConn.WriteMessage(msg); err != nil {
		c.Errorf("cannot write message: %w", err)
	}
}

func (c *session) Get(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error) {
	req, err := c.coapConn.NewGetRequest(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	return c.Do(req, "")
}

func (c *session) Observe(ctx context.Context, path string, observeFunc func(req *pool.Message), opts ...message.Option) (observation.Observation, error) {
	req, err := c.coapConn.NewObserveRequest(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	t := time.Now()
	obs, err := c.coapConn.DoObserve(req, observeFunc)
	logger := c.getLogger()
	if err == nil && !WantToLog(codes.Content, logger) {
		return obs, err
	}
	logger = logger.With(log.StartTimeKey, t, log.DurationMSKey, log.DurationToMilliseconds(time.Since(t)))
	logger = c.loggerWithRequestResponse(logger, req, nil)
	if err != nil {
		_ = logger.LogAndReturnError(fmt.Errorf("create observation to the device fails with error: %w", err))
	} else if WantToLog(codes.Content, logger) {
		DefaultCodeToLevel(codes.Content, logger)("finished creating observation to the device")
	}
	return obs, err
}

func (c *session) ReleaseMessage(m *pool.Message) {
	c.server.messagePool.ReleaseMessage(m)
}

func (c *session) do(req *pool.Message) (*pool.Message, error) {
	path, _ := req.Path()
	ctx, span := otelcoap.Start(req.Context(), path, req.Code().String(), otelcoap.WithTracerProvider(c.server.tracerProvider), otelcoap.WithSpanOptions(trace.WithSpanKind(trace.SpanKindClient)))
	defer span.End()
	span.SetAttributes(semconv.NetPeerNameKey.String(c.deviceID()))

	otelcoap.MessageSentEvent(ctx, req)

	resp, err := c.coapConn.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return nil, err
	}
	otelcoap.MessageReceivedEvent(ctx, resp)
	span.SetAttributes(otelcoap.StatusCodeAttr(resp.Code()))

	return resp, nil
}

func (c *session) Do(req *pool.Message, correlationID string) (*pool.Message, error) {
	t := time.Now()
	resp, err := c.do(req)
	logger := c.getLogger()
	if err == nil && resp != nil && !WantToLog(resp.Code(), logger) {
		return resp, err
	}
	logger = logger.With(log.StartTimeKey, t, log.DurationMSKey, log.DurationToMilliseconds(time.Since(t)))
	logger = c.loggerWithRequestResponse(logger, req, resp)
	if correlationID != "" {
		logger = logger.With(log.CorrelationIDKey, correlationID)
	}
	if err != nil {
		_ = logger.LogAndReturnError(fmt.Errorf("finished unary call to the device with error: %w", err))
	} else if WantToLog(resp.Code(), logger) {
		DefaultCodeToLevel(resp.Code(), logger)(fmt.Sprintf("finished unary call to the device with code %v", resp.Code()))
	}
	return resp, err
}

func (c *session) GetDevicesMetadata(ctx context.Context, in *pb.GetDevicesMetadataRequest, opts ...grpc.CallOption) (pb.GrpcGateway_GetDevicesMetadataClient, error) {
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		return nil, err
	}
	ctx = kitNetGrpc.CtxWithToken(ctx, authCtx.GetAccessToken())
	return c.server.rdClient.GetDevicesMetadata(ctx, in, opts...)
}

func (c *session) GetResourceLinks(ctx context.Context, in *pb.GetResourceLinksRequest, opts ...grpc.CallOption) (pb.GrpcGateway_GetResourceLinksClient, error) {
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		return nil, err
	}
	ctx = kitNetGrpc.CtxWithToken(ctx, authCtx.GetAccessToken())
	return c.server.rdClient.GetResourceLinks(ctx, in, opts...)
}

func (c *session) UnpublishResourceLinks(ctx context.Context, in *commands.UnpublishResourceLinksRequest, opts ...grpc.CallOption) (*commands.UnpublishResourceLinksResponse, error) {
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		return nil, err
	}
	ctx = kitNetGrpc.CtxWithToken(ctx, authCtx.GetAccessToken())
	return c.server.raClient.UnpublishResourceLinks(ctx, in, opts...)
}

func (c *session) RemoteAddr() net.Addr {
	return c.coapConn.RemoteAddr()
}

func (c *session) Context() context.Context {
	return c.coapConn.Context()
}

func (c *session) cancelResourceSubscription(token string) (bool, error) {
	s, ok := c.resourceSubscriptions.PullOut(token)
	if !ok {
		return false, nil
	}
	sub := s.(*resourceSubscription)

	if err := sub.Close(); err != nil {
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
func (c *session) onGetResourceContent(ctx context.Context, deviceID, href string, notification *pool.Message) error {
	cannotGetResourceContentError := func(deviceID, href string, err error) error {
		return fmt.Errorf("cannot get resource /%v%v content: %w", deviceID, href, err)
	}
	notification.Hijack()
	err := c.server.taskQueue.Submit(func() {
		defer c.server.messagePool.ReleaseMessage(notification)
		if notification.Code() == codes.NotFound {
			c.unpublishResourceLinks(c.getUserAuthorizedContext(ctx), []string{href}, nil)
		}
		err2 := c.notifyContentChanged(deviceID, href, false, notification)
		if err2 != nil {
			// hub is out of sync with the device, for recovery, the device is disconnected from the hub
			c.Close()
			c.Errorf("%w", cannotGetResourceContentError(deviceID, href, err2))
			return
		}
		obs, err := c.getDeviceObserver(c.Context())
		if err == nil {
			obs.ResourceHasBeenSynchronized(ctx, href)
		}
	})
	if err != nil {
		defer c.server.messagePool.ReleaseMessage(notification)
		// hub is out of sync with the device, for recovery, the device is disconnected from the hub
		c.Close()
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
func (c *session) onObserveResource(ctx context.Context, deviceID, href string, batch bool, notification *pool.Message) error {
	cannotObserResourceError := func(err error) error {
		return fmt.Errorf("cannot handle resource observation: %w", err)
	}
	notification.Hijack()
	err := c.server.taskQueue.SubmitForOneWorker(resource.GetInstanceID(deviceID+href), func() {
		defer c.server.messagePool.ReleaseMessage(notification)
		if notification.Code() == codes.NotFound {
			c.unpublishResourceLinks(c.getUserAuthorizedContext(notification.Context()), []string{href}, nil)
		}
		err2 := c.notifyContentChanged(deviceID, href, batch, notification)
		if err2 != nil {
			// hub is out of sync with the device, for recovery, the device is disconnected from the hub
			c.Close()
			c.Errorf("%w", cannotObserResourceError(err2))
			return
		}
		obs, err := c.getDeviceObserver(c.Context())
		if err == nil {
			obs.ResourceHasBeenSynchronized(ctx, href)
		}
	})
	if err != nil {
		defer c.server.messagePool.ReleaseMessage(notification)
		// hub is out of sync with the device, for recovery, the device is disconnected from the hub
		c.Close()
		return cannotObserResourceError(err)
	}
	return nil
}

// Close closes coap connection
func (c *session) Close() {
	err := c.coapConn.Close()
	if err != nil {
		c.Errorf("cannot close client: %w", err)
	}
}

func (c *session) cancelResourceSubscriptions(wantWait bool) {
	resourceSubscriptions := c.resourceSubscriptions.PullOutAll()
	for _, v := range resourceSubscriptions {
		o, ok := grpcClient.ToResourceSubscription(v, true)
		if !ok {
			continue
		}
		wait, err := o.Cancel()
		if err != nil {
			c.Errorf("cannot cancel resource subscription: %w", err)
		} else if wantWait {
			wait()
		}
	}
}

func (c *session) CleanUp(resetAuthContext bool) *authorizationContext {
	authCtx, _ := c.GetAuthorizationContext()
	c.server.devicesStatusUpdater.Remove(c)
	if err := c.closeDeviceObserver(c.Context()); err != nil {
		c.Errorf("cleanUp error: failed to close observer for device %v: %w", authCtx.GetDeviceID(), err)
	}
	c.cancelResourceSubscriptions(false)
	if err := c.closeDeviceSubscriber(); err != nil {
		c.Errorf("cleanUp error: failed to close device %v subscription: %w", authCtx.GetDeviceID(), err)
	}
	c.unsubscribeFromDeviceEvents()

	if resetAuthContext {
		return c.SetAuthorizationContext(nil)
	}
	// we cannot reset authorizationContext need token (eg signOff)
	return authCtx
}

// OnClose is invoked when the coap connection was closed.
func (c *session) OnClose() {
	authCtx, _ := c.GetAuthorizationContext()
	if authCtx.GetDeviceID() != "" {
		// don't log health check connection
		c.Debugf("close device connection")
	}
	oldAuthCtx := c.CleanUp(false)

	if oldAuthCtx.GetDeviceID() != "" {
		c.server.expirationClientCache.Delete(oldAuthCtx.GetDeviceID())
		ctx, cancel := context.WithTimeout(context.Background(), c.server.config.APIs.COAP.KeepAlive.Timeout)
		defer cancel()
		_, err := c.server.raClient.UpdateDeviceMetadata(kitNetGrpc.CtxWithToken(ctx, oldAuthCtx.GetAccessToken()), &commands.UpdateDeviceMetadataRequest{
			DeviceId: authCtx.GetDeviceID(),
			Update: &commands.UpdateDeviceMetadataRequest_Connection{
				Connection: &commands.Connection{
					Status: commands.Connection_OFFLINE,
				},
			},
			CommandMetadata: &commands.CommandMetadata{
				Sequence:     c.coapConn.Sequence(),
				ConnectionId: c.RemoteAddr().String(),
			},
		})
		if err != nil {
			// Device will be still reported as online and it can fix his state by next calls online, offline commands.
			c.Errorf("DeviceID %v: cannot handle sign out: cannot update cloud device status: %w", oldAuthCtx.GetDeviceID(), err)
		}
	}
}

func (c *session) SetAuthorizationContext(authCtx *authorizationContext) (oldDeviceID *authorizationContext) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	oldAuthContext := c.authCtx
	c.authCtx = authCtx
	return oldAuthContext
}

func (c *session) GetAuthorizationContext() (*authorizationContext, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.authCtx, c.authCtx.IsValid()
}

func (c *session) batchNotifyContentChanged(ctx context.Context, deviceID string, notification *pool.Message) error {
	batch, err := coapconv.NewNotifyResourceChangedRequestsFromBatchResourceDiscovery(deviceID, c.RemoteAddr().String(), notification)
	if err != nil {
		return err
	}
	_, err = c.server.raClient.BatchNotifyResourceChanged(ctx, &commands.BatchNotifyResourceChangedRequest{
		Batch: batch,
	})
	return err
}

func (c *session) notifyContentChanged(deviceID, href string, batch bool, notification *pool.Message) error {
	notifyError := func(deviceID, href string, err error) error {
		return fmt.Errorf("cannot notify resource /%v%v content changed: %w", deviceID, href, err)
	}
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		return notifyError(deviceID, href, err)
	}
	if _, err = notification.Observe(); err == nil {
		// we want to log only observations
		c.logNotificationFromClient(href, notification)
	}
	ctx := kitNetGrpc.CtxWithToken(c.Context(), authCtx.GetAccessToken())
	if batch && href == resources.ResourceURI {
		err = c.batchNotifyContentChanged(ctx, deviceID, notification)
		if err != nil {
			return notifyError(deviceID, href, err)
		}
		return nil
	}
	_, err = c.server.raClient.NotifyResourceChanged(ctx, coapconv.NewNotifyResourceChangedRequest(commands.NewResourceID(deviceID, href), c.RemoteAddr().String(), notification))
	if err != nil {
		return notifyError(deviceID, href, err)
	}
	return nil
}

func (c *session) sendErrorConfirmResourceUpdate(ctx context.Context, deviceID, href, correlationID string, code codes.Code, errToSend error) {
	resp := c.server.messagePool.AcquireMessage(ctx)
	defer c.server.messagePool.ReleaseMessage(resp)
	resp.SetContentFormat(message.TextPlain)
	resp.SetBody(bytes.NewReader([]byte(errToSend.Error())))
	resp.SetCode(code)

	request := coapconv.NewConfirmResourceUpdateRequest(commands.NewResourceID(deviceID, href), correlationID, c.RemoteAddr().String(), resp)
	_, err := c.server.raClient.ConfirmResourceUpdate(ctx, request)
	if err != nil {
		c.Errorf("cannot send error via confirm resource update: %w", err)
	}
}

func setDeviceIDToTracerSpan(ctx context.Context, deviceID string) {
	if deviceID != "" {
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(semconv.NetPeerNameKey.String(deviceID))
	}
}

func (c *session) updateStatusResource(ctx context.Context, sendConfirmCtx context.Context, event *events.ResourceUpdatePending) error {
	msg := c.server.messagePool.AcquireMessage(ctx)
	msg.SetCode(codes.MethodNotAllowed)
	msg.SetSequence(c.coapConn.Sequence())
	defer c.server.messagePool.ReleaseMessage(msg)
	request := coapconv.NewConfirmResourceUpdateRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), c.RemoteAddr().String(), msg)
	_, err := c.server.raClient.ConfirmResourceUpdate(sendConfirmCtx, request)
	if err != nil {
		return err
	}
	return nil
}

func (c *session) UpdateResource(ctx context.Context, event *events.ResourceUpdatePending) error {
	setDeviceIDToTracerSpan(ctx, c.deviceID())
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		c.Close()
		return fmt.Errorf("cannot update resource /%v%v: %w", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), err)
	}
	sendConfirmCtx := authCtx.ToContext(ctx)
	if event.GetResourceId().GetHref() == commands.StatusHref {
		return c.updateStatusResource(ctx, sendConfirmCtx, event)
	}

	coapCtx, cancel := context.WithTimeout(ctx, c.server.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceUpdateRequest(coapCtx, c.server.messagePool, event)
	if err != nil {
		c.sendErrorConfirmResourceUpdate(sendConfirmCtx, event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), event.GetAuditContext().GetCorrelationId(), codes.BadRequest, err)
		return err
	}
	defer c.server.messagePool.ReleaseMessage(req)

	resp, err := c.Do(req, event.GetAuditContext().GetCorrelationId())
	if err != nil {
		c.sendErrorConfirmResourceUpdate(sendConfirmCtx, event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), event.GetAuditContext().GetCorrelationId(), codes.ServiceUnavailable, err)
		return err
	}
	defer c.server.messagePool.ReleaseMessage(resp)

	if resp.Code() == codes.NotFound {
		c.unpublishResourceLinks(c.getUserAuthorizedContext(ctx), []string{event.GetResourceId().GetHref()}, nil)
	}

	request := coapconv.NewConfirmResourceUpdateRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), c.RemoteAddr().String(), resp)
	_, err = c.server.raClient.ConfirmResourceUpdate(sendConfirmCtx, request)
	if err != nil {
		return err
	}

	return nil
}

func (c *session) sendErrorConfirmResourceRetrieve(ctx context.Context, deviceID, href, correlationID string, code codes.Code, errToSend error) {
	resp := c.server.messagePool.AcquireMessage(ctx)
	defer c.server.messagePool.ReleaseMessage(resp)
	resp.SetContentFormat(message.TextPlain)
	resp.SetBody(bytes.NewReader([]byte(errToSend.Error())))
	resp.SetCode(code)
	request := coapconv.NewConfirmResourceRetrieveRequest(commands.NewResourceID(deviceID, href), correlationID, c.RemoteAddr().String(), resp)
	_, err := c.server.raClient.ConfirmResourceRetrieve(ctx, request)
	if err != nil {
		c.Errorf("cannot send error confirm resource retrieve: %w", err)
	}
}

func (c *session) retrieveStatusResource(ctx context.Context, sendConfirmCtx context.Context, event *events.ResourceRetrievePending) error {
	msg := c.server.messagePool.AcquireMessage(ctx)
	msg.SetCode(codes.Content)
	msg.SetSequence(c.coapConn.Sequence())
	defer c.server.messagePool.ReleaseMessage(msg)

	request := coapconv.NewConfirmResourceRetrieveRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), c.RemoteAddr().String(), msg)
	_, err := c.server.raClient.ConfirmResourceRetrieve(sendConfirmCtx, request)
	if err != nil {
		return err
	}
	return nil
}

func (c *session) RetrieveResource(ctx context.Context, event *events.ResourceRetrievePending) error {
	setDeviceIDToTracerSpan(ctx, c.deviceID())
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		c.Close()
		return fmt.Errorf("cannot retrieve resource /%v%v: %w", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), err)
	}
	sendConfirmCtx := authCtx.ToContext(ctx)
	if event.GetResourceId().GetHref() == commands.StatusHref {
		return c.retrieveStatusResource(ctx, sendConfirmCtx, event)
	}

	coapCtx, cancel := context.WithTimeout(ctx, c.server.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceRetrieveRequest(coapCtx, c.server.messagePool, event)
	if err != nil {
		c.sendErrorConfirmResourceRetrieve(sendConfirmCtx, event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), event.GetAuditContext().GetCorrelationId(), codes.BadRequest, err)
		return err
	}
	defer c.server.messagePool.ReleaseMessage(req)

	resp, err := c.Do(req, event.GetAuditContext().GetCorrelationId())
	if err != nil {
		c.sendErrorConfirmResourceRetrieve(sendConfirmCtx, event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), event.GetAuditContext().GetCorrelationId(), codes.ServiceUnavailable, err)
		return err
	}
	defer c.server.messagePool.ReleaseMessage(resp)

	if resp.Code() == codes.NotFound {
		c.unpublishResourceLinks(c.getUserAuthorizedContext(ctx), []string{event.GetResourceId().GetHref()}, nil)
	}

	request := coapconv.NewConfirmResourceRetrieveRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), c.RemoteAddr().String(), resp)
	_, err = c.server.raClient.ConfirmResourceRetrieve(sendConfirmCtx, request)
	if err != nil {
		return err
	}

	return nil
}

func (c *session) sendErrorConfirmResourceDelete(ctx context.Context, deviceID, href, correlationID string, code codes.Code, errToSend error) {
	resp := c.server.messagePool.AcquireMessage(ctx)
	defer c.server.messagePool.ReleaseMessage(resp)
	resp.SetContentFormat(message.TextPlain)
	resp.SetBody(bytes.NewReader([]byte(errToSend.Error())))
	resp.SetCode(code)
	request := coapconv.NewConfirmResourceDeleteRequest(commands.NewResourceID(deviceID, href), correlationID, c.RemoteAddr().String(), resp)
	_, err := c.server.raClient.ConfirmResourceDelete(ctx, request)
	if err != nil {
		c.Errorf("cannot send error via confirm resource delete: %w", err)
	}
}

func (c *session) deleteStatusResource(ctx context.Context, sendConfirmCtx context.Context, event *events.ResourceDeletePending) error {
	msg := c.server.messagePool.AcquireMessage(ctx)
	msg.SetCode(codes.Forbidden)
	msg.SetSequence(c.coapConn.Sequence())
	defer c.server.messagePool.ReleaseMessage(msg)

	request := coapconv.NewConfirmResourceDeleteRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), c.RemoteAddr().String(), msg)
	_, err := c.server.raClient.ConfirmResourceDelete(sendConfirmCtx, request)
	if err != nil {
		return err
	}
	return nil
}

func (c *session) DeleteResource(ctx context.Context, event *events.ResourceDeletePending) error {
	setDeviceIDToTracerSpan(ctx, c.deviceID())
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		c.Close()
		return fmt.Errorf("cannot delete resource /%v%v: %w", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), err)
	}
	sendConfirmCtx := authCtx.ToContext(ctx)
	if event.GetResourceId().GetHref() == commands.StatusHref {
		return c.deleteStatusResource(ctx, sendConfirmCtx, event)
	}

	coapCtx, cancel := context.WithTimeout(ctx, c.server.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceDeleteRequest(coapCtx, c.server.messagePool, event)
	if err != nil {
		c.sendErrorConfirmResourceDelete(sendConfirmCtx, event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), event.GetAuditContext().GetCorrelationId(), codes.BadRequest, err)
		return err
	}
	defer c.server.messagePool.ReleaseMessage(req)

	resp, err := c.Do(req, event.GetAuditContext().GetCorrelationId())
	if err != nil {
		c.sendErrorConfirmResourceDelete(sendConfirmCtx, event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), event.GetAuditContext().GetCorrelationId(), codes.ServiceUnavailable, err)
		return err
	}
	defer c.server.messagePool.ReleaseMessage(resp)

	if resp.Code() == codes.NotFound {
		c.unpublishResourceLinks(c.getUserAuthorizedContext(ctx), []string{event.GetResourceId().GetHref()}, nil)
	}

	request := coapconv.NewConfirmResourceDeleteRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), c.RemoteAddr().String(), resp)
	_, err = c.server.raClient.ConfirmResourceDelete(sendConfirmCtx, request)
	if err != nil {
		return err
	}

	return nil
}

func (c *session) getUserAuthorizedContext(ctx context.Context) context.Context {
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		c.Errorf("unable to load authorization context: %w", err)
		return nil
	}

	return authCtx.ToContext(ctx)
}

func (c *session) unpublishResourceLinks(ctx context.Context, hrefs []string, instanceIDs []int64) []string {
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		c.Errorf("unable to load authorization context during resource links publish for device: %w", err)
		return nil
	}

	logUnpublishError := func(err error) {
		c.Errorf("error occurred during resource links unpublish for device %v: %w", authCtx.GetDeviceID(), err)
	}
	resp, err := c.server.raClient.UnpublishResourceLinks(ctx, &commands.UnpublishResourceLinksRequest{
		Hrefs:       hrefs,
		InstanceIds: instanceIDs,
		DeviceId:    authCtx.GetDeviceID(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: c.RemoteAddr().String(),
			Sequence:     c.coapConn.Sequence(),
		},
	})
	if err != nil {
		// unpublish resource is not critical -> resource can be still accessible
		// next resource update will return 'not found' what triggers a publish again
		logUnpublishError(err)
		return nil
	}

	if len(resp.UnpublishedHrefs) == 0 {
		return nil
	}

	observer, err := c.getDeviceObserver(ctx)
	if err != nil {
		logUnpublishError(err)
		return resp.UnpublishedHrefs
	}
	observer.RemovePublishedResources(ctx, resp.UnpublishedHrefs)
	return resp.UnpublishedHrefs
}

func (c *session) sendErrorConfirmResourceCreate(ctx context.Context, resourceID *commands.ResourceId, correlationID string, code codes.Code, errToSend error) {
	resp := c.server.messagePool.AcquireMessage(ctx)
	defer c.server.messagePool.ReleaseMessage(resp)
	resp.SetContentFormat(message.TextPlain)
	resp.SetBody(bytes.NewReader([]byte(errToSend.Error())))
	resp.SetCode(code)
	request := coapconv.NewConfirmResourceCreateRequest(resourceID, correlationID, c.RemoteAddr().String(), resp)
	_, err := c.server.raClient.ConfirmResourceCreate(ctx, request)
	if err != nil {
		c.Errorf("cannot send error via confirm resource create: %w", err)
	}
}

func (c *session) createStatusResource(ctx context.Context, sendConfirmCtx context.Context, event *events.ResourceCreatePending) error {
	msg := c.server.messagePool.AcquireMessage(ctx)
	msg.SetCode(codes.Forbidden)
	msg.SetSequence(c.coapConn.Sequence())
	defer c.server.messagePool.ReleaseMessage(msg)
	request := coapconv.NewConfirmResourceCreateRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), c.RemoteAddr().String(), msg)
	_, err := c.server.raClient.ConfirmResourceCreate(sendConfirmCtx, request)
	if err != nil {
		return err
	}
	return nil
}

func (c *session) CreateResource(ctx context.Context, event *events.ResourceCreatePending) error {
	setDeviceIDToTracerSpan(ctx, c.deviceID())
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		c.Close()
		return fmt.Errorf("cannot create resource /%v%v: %w", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref(), err)
	}
	sendConfirmCtx := authCtx.ToContext(ctx)
	if event.GetResourceId().GetHref() == commands.StatusHref {
		return c.createStatusResource(ctx, sendConfirmCtx, event)
	}

	coapCtx, cancel := context.WithTimeout(ctx, c.server.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()
	req, err := coapconv.NewCoapResourceCreateRequest(coapCtx, c.server.messagePool, event)
	if err != nil {
		c.sendErrorConfirmResourceCreate(sendConfirmCtx, event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), codes.BadRequest, err)
		return err
	}
	defer c.server.messagePool.ReleaseMessage(req)

	resp, err := c.Do(req, event.GetAuditContext().GetCorrelationId())
	if err != nil {
		c.sendErrorConfirmResourceCreate(sendConfirmCtx, event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), codes.ServiceUnavailable, err)
		return err
	}
	defer c.server.messagePool.ReleaseMessage(resp)

	if resp.Code() == codes.NotFound {
		c.unpublishResourceLinks(c.getUserAuthorizedContext(ctx), []string{event.GetResourceId().GetHref()}, nil)
	}

	request := coapconv.NewConfirmResourceCreateRequest(event.GetResourceId(), event.GetAuditContext().GetCorrelationId(), c.RemoteAddr().String(), resp)
	_, err = c.server.raClient.ConfirmResourceCreate(sendConfirmCtx, request)
	if err != nil {
		return err
	}

	return nil
}

func (c *session) OnDeviceSubscriberReconnectError(err error) {
	auth, _ := c.GetAuthorizationContext()
	deviceID := auth.GetDeviceID()
	c.Errorf("cannot reconnect device %v subscriber to resource directory or eventbus - closing the device connection: %w", deviceID, err)
	c.Close()
	logCloseDeviceSubscriberError := func(err error) {
		c.Errorf("failed to close device %v subscription: %w", auth.GetDeviceID(), err)
	}
	if err = c.server.taskQueue.Submit(func() {
		if errC := c.closeDeviceSubscriber(); errC != nil {
			logCloseDeviceSubscriberError(errC)
		}
	}); err != nil {
		logCloseDeviceSubscriberError(err)
	}
}

func (c *session) GetContext() context.Context {
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		return c.Context()
	}
	return authCtx.ToContext(c.Context())
}

func (c *session) confirmDeviceMetadataUpdate(ctx context.Context, event *events.DeviceMetadataUpdatePending) error {
	_, err := c.server.raClient.ConfirmDeviceMetadataUpdate(ctx, &commands.ConfirmDeviceMetadataUpdateRequest{
		DeviceId:      event.GetDeviceId(),
		CorrelationId: event.GetAuditContext().GetCorrelationId(),
		Confirm: &commands.ConfirmDeviceMetadataUpdateRequest_TwinEnabled{
			TwinEnabled: event.GetTwinEnabled(),
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: c.RemoteAddr().String(),
			Sequence:     c.coapConn.Sequence(),
		},
		Status: commands.Status_OK,
	})
	return err
}

func (c *session) UpdateDeviceMetadata(ctx context.Context, event *events.DeviceMetadataUpdatePending) error {
	setDeviceIDToTracerSpan(ctx, c.deviceID())
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		c.Close()
		return fmt.Errorf("cannot update device('%v') metadata: %w", event.GetDeviceId(), err)
	}
	if _, ok := event.GetUpdatePending().(*events.DeviceMetadataUpdatePending_TwinEnabled); !ok {
		return nil
	}
	sendConfirmCtx := authCtx.ToContext(ctx)

	var errObs error
	var previous bool
	if event.GetTwinEnabled() {
		// if twin is enabled, we need to first update twin synchronization state to sync out
		// and then synchronization state will be updated by other replaceDeviceObserverWithDeviceTwin
		err = c.confirmDeviceMetadataUpdate(sendConfirmCtx, event)
		previous, errObs = c.replaceDeviceObserverWithDeviceTwin(sendConfirmCtx, event.GetTwinEnabled())
	} else {
		// if twin is disabled, we to stop observation resources to disable all update twin synchronization state
		previous, errObs = c.replaceDeviceObserverWithDeviceTwin(sendConfirmCtx, event.GetTwinEnabled())
		// and then we need to update twin synchronization state to disabled
		err = c.confirmDeviceMetadataUpdate(sendConfirmCtx, event)
	}
	if errObs != nil {
		c.Close()
		return fmt.Errorf("cannot update device('%v') metadata: %w", event.GetDeviceId(), errObs)
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		_, errObs := c.replaceDeviceObserverWithDeviceTwin(sendConfirmCtx, previous)
		if errObs != nil {
			c.Close()
			c.Errorf("update device('%v') metadata error: %w", event.GetDeviceId(), errObs)
		}
	}
	return err
}

func (c *session) ValidateToken(ctx context.Context, token string) (pkgJwt.Claims, error) {
	return c.server.ValidateToken(ctx, token)
}

func (c *session) subscribeToDeviceEvents(owner string, onEvent func(e *idEvents.Event)) error {
	close, err := c.server.ownerCache.Subscribe(owner, onEvent)
	if err != nil {
		return err
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.closeEventSubscriptions = close
	return nil
}

func (c *session) unsubscribeFromDeviceEvents() {
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

func (c *session) ResolveDeviceID(claim pkgJwt.Claims, paramDeviceID string) string {
	if c.server.config.APIs.COAP.Authorization.DeviceIDClaim != "" {
		return claim.DeviceID(c.server.config.APIs.COAP.Authorization.DeviceIDClaim)
	}
	if c.server.config.APIs.COAP.TLS.IsEnabled() && c.server.config.APIs.COAP.TLS.Embedded.ClientCertificateRequired {
		return c.tlsDeviceID
	}
	return paramDeviceID
}

func (c *session) UpdateTwinSynchronizationStatus(ctx context.Context, deviceID string, state commands.TwinSynchronization_State, t time.Time) error {
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		c.Close()
		return fmt.Errorf("cannot update twin synchronization %v to %v: %w", deviceID, state, err)
	}

	var startAt, finishedAt int64
	switch state {
	case commands.TwinSynchronization_SYNCING:
		startAt = t.UnixNano()
	case commands.TwinSynchronization_IN_SYNC:
		finishedAt = t.UnixNano()
	}
	ctx = authCtx.ToContext(ctx)
	_, err = c.server.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
		DeviceId: deviceID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: c.RemoteAddr().String(),
			Sequence:     c.coapConn.Sequence(),
		},
		Update: &commands.UpdateDeviceMetadataRequest_TwinSynchronization{
			TwinSynchronization: &commands.TwinSynchronization{
				State:     state,
				SyncingAt: startAt,
				InSyncAt:  finishedAt,
			},
		},
	})
	return err
}
