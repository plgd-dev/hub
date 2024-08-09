package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/pion/dtls/v2"
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
	"github.com/plgd-dev/hub/v2/pkg/net/coap"
	coapService "github.com/plgd-dev/hub/v2/pkg/net/coap/service"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry/otelcoap"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/future"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	pbRD "github.com/plgd-dev/hub/v2/resource-directory/pb"
	kitSync "github.com/plgd-dev/kit/v2/sync"
	otelCodes "go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
)

type authorizationContext struct {
	Expire      time.Time
	DeviceID    string
	AccessToken string
	UserID      string
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
		jwtClaims, err := pkgJwt.ParseToken(a.AccessToken)
		if err == nil {
			return jwtClaims
		}
	}
	return make(pkgJwt.Claims)
}

func (a *authorizationContext) IsValid() error {
	if a == nil {
		return errors.New("invalid authorization context")
	}
	if a.AccessToken == "" {
		return errors.New("invalid access token")
	}
	if !a.Expire.IsZero() && time.Now().UnixNano() > a.Expire.UnixNano() {
		return errors.New("token is expired")
	}
	return nil
}

func (a *authorizationContext) ToContext(ctx context.Context) context.Context {
	return kitNetGrpc.CtxWithToken(ctx, a.GetAccessToken())
}

// session a setup of connection
type session struct {
	tlsValidUntil         time.Time
	coapConn              mux.Conn
	server                *Service
	resourceSubscriptions *kitSync.Map
	exchangeCache         *ExchangeCache
	refreshCache          *RefreshCache
	tlsDeviceID           string
	private               struct { // guarded by mutex
		mutex                   sync.Mutex
		authCtx                 *authorizationContext
		deviceSubscriber        *grpcClient.DeviceSubscriber
		deviceObserver          *future.Future
		closeEventSubscriptions func()
	}

	// blockSignOff is used to block sign off until all commands from hub are finished.
	// eg: factory reset was send via /oic/mnt resource and sign off are called in parallel.
	blockSignOff *semaphore.Weighted
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
		blockSignOff:          semaphore.NewWeighted(math.MaxInt64),
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

func (c *session) GetApplicationProtocol() commands.Connection_Protocol {
	switch c.coapConn.NetConn().(type) {
	case *tls.Conn:
		return commands.Connection_COAPS_TCP
	case *dtls.Conn:
		return commands.Connection_COAPS
	case *net.TCPConn:
		return commands.Connection_COAP_TCP
	case *net.UDPConn:
		return commands.Connection_COAP
	}
	return commands.Connection_UNKNOWN
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

	otelcoap.MessageSentEvent(ctx, otelcoap.MakeMessage(req))

	resp, err := c.coapConn.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return nil, err
	}
	otelcoap.MessageReceivedEvent(ctx, otelcoap.MakeMessage(resp))
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

func (c *session) GetLatestDeviceETags(ctx context.Context, in *pbRD.GetLatestDeviceETagsRequest, opts ...grpc.CallOption) (*pbRD.GetLatestDeviceETagsResponse, error) {
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		return nil, err
	}
	ctx = kitNetGrpc.CtxWithToken(ctx, authCtx.GetAccessToken())
	return c.server.rdClient.GetLatestDeviceETags(ctx, in, opts...)
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
func (c *session) onGetResourceContent(ctx context.Context, deviceID, href string, resourceTypes []string, notification *pool.Message) error {
	cannotGetResourceContentError := func(deviceID, href string, err error) error {
		return fmt.Errorf("cannot get resource /%v%v content: %w", deviceID, href, err)
	}
	notification.Hijack()
	x := struct {
		ctx                           context.Context
		notification                  *pool.Message
		deviceID                      string
		href                          string
		resourceTypes                 []string
		c                             *session
		cannotGetResourceContentError func(deviceID, href string, err error) error
	}{
		ctx:                           ctx,
		notification:                  notification,
		deviceID:                      deviceID,
		href:                          href,
		resourceTypes:                 resourceTypes,
		c:                             c,
		cannotGetResourceContentError: cannotGetResourceContentError,
	}
	err := c.server.taskQueue.Submit(func() {
		defer x.c.server.messagePool.ReleaseMessage(x.notification)
		if x.notification.Code() == codes.NotFound {
			x.c.unpublishResourceLinks(x.c.getUserAuthorizedContext(x.ctx), []string{x.href}, nil)
		}
		err2 := x.c.notifyContentChanged(x.deviceID, x.href, x.resourceTypes, false, x.notification)
		if err2 != nil {
			// hub is out of sync with the device, for recovery, the device is disconnected from the hub
			x.c.Close()
			x.c.Errorf("%w", x.cannotGetResourceContentError(x.deviceID, x.href, err2))
			return
		}
		obs, ok, _ := x.c.getDeviceObserver(x.c.Context())
		if ok {
			obs.ResourceHasBeenSynchronized(x.ctx, x.href)
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
func (c *session) onObserveResource(ctx context.Context, deviceID, href string, resourceTypes []string, batch bool, notification *pool.Message) error {
	cannotObserResourceError := func(err error) error {
		return fmt.Errorf("cannot handle resource observation: %w", err)
	}
	notification.Hijack()
	x := struct {
		ctx                      context.Context
		notification             *pool.Message
		deviceID                 string
		href                     string
		resourceTypes            []string
		c                        *session
		cannotObserResourceError func(err error) error
		batch                    bool
	}{
		ctx:                      ctx,
		notification:             notification,
		deviceID:                 deviceID,
		href:                     href,
		resourceTypes:            resourceTypes,
		c:                        c,
		cannotObserResourceError: cannotObserResourceError,
		batch:                    batch,
	}
	err := c.server.taskQueue.SubmitForOneWorker(resource.GetInstanceID(deviceID+href), func() {
		defer x.c.server.messagePool.ReleaseMessage(x.notification)
		if x.notification.Code() == codes.NotFound {
			x.c.unpublishResourceLinks(x.c.getUserAuthorizedContext(x.notification.Context()), []string{x.href}, nil)
		}
		err2 := x.c.notifyContentChanged(x.deviceID, x.href, x.resourceTypes, x.batch, x.notification)
		if err2 != nil {
			// hub is out of sync with the device, for recovery, the device is disconnected from the hub
			x.c.Close()
			x.c.Errorf("%w", x.cannotObserResourceError(err2))
			return
		}
		obs, ok, _ := x.c.getDeviceObserver(x.c.Context())
		if ok {
			obs.ResourceHasBeenSynchronized(x.ctx, x.href)
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
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	oldAuthContext := c.private.authCtx
	c.private.authCtx = authCtx
	return oldAuthContext
}

func (c *session) GetAuthorizationContext() (*authorizationContext, error) {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	return c.private.authCtx, c.private.authCtx.IsValid()
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

func (c *session) getLocalEndpoints() []string {
	localEndpoints, err := coap.GetEndpointsFromDeviceResource(c.Context(), c)
	if err != nil {
		c.getLogger().Warnf("cannot get local endpoints: %v", err)
		return nil
	}
	c.getLogger().With(log.LocalEndpointsKey, localEndpoints).Debugf("local endpoints retrieval successful.")
	return localEndpoints
}

func (c *session) notifyContentChanged(deviceID, href string, resourceTypes []string, batch bool, notification *pool.Message) error {
	if !c.blockSignOff.TryAcquire(1) {
		c.getLogger().Debugf("cannot notify resource /%v%v content changed: signOff processing", deviceID, href)
		return nil
	}
	defer c.blockSignOff.Release(1)
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
	// the content of the resource is up to date, codes.Valid is used to indicate that the resource has not changed for GET with the etag.
	bodySize, err := notification.BodySize()
	if err != nil {
		return notifyError(deviceID, href, err)
	}
	if notification.Code() == codes.Valid && bodySize == 0 {
		c.Debugf("resource /%v%v content is up to date", deviceID, href)
		return nil
	}
	ctx := kitNetGrpc.CtxWithToken(c.Context(), authCtx.GetAccessToken())
	if batch && href == resources.ResourceURI {
		err = c.batchNotifyContentChanged(ctx, deviceID, notification)
		if err != nil {
			return notifyError(deviceID, href, err)
		}
		return nil
	}
	_, err = c.server.raClient.NotifyResourceChanged(ctx, coapconv.NewNotifyResourceChangedRequest(commands.NewResourceID(deviceID, href), resourceTypes, c.RemoteAddr().String(), notification))
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
	if !c.blockSignOff.TryAcquire(1) {
		return fmt.Errorf("cannot update resource /%v%v: signOff processing", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref())
	}
	defer c.blockSignOff.Release(1)
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
	if !c.blockSignOff.TryAcquire(1) {
		return fmt.Errorf("cannot retrieve resource /%v%v: signOff processing", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref())
	}
	defer c.blockSignOff.Release(1)
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
	if !c.blockSignOff.TryAcquire(1) {
		return fmt.Errorf("cannot delete resource /%v%v: signOff processing", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref())
	}
	defer c.blockSignOff.Release(1)
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

	if len(resp.GetUnpublishedHrefs()) == 0 {
		return nil
	}

	observer, ok, err := c.getDeviceObserver(ctx)
	if err != nil {
		logUnpublishError(err)
		return resp.GetUnpublishedHrefs()
	}
	if !ok {
		logUnpublishError(errors.New("device observer not found"))
		return resp.GetUnpublishedHrefs()
	}
	observer.RemovePublishedResources(ctx, resp.GetUnpublishedHrefs())
	return resp.GetUnpublishedHrefs()
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
	if !c.blockSignOff.TryAcquire(1) {
		return fmt.Errorf("cannot create resource /%v%v: signOff processing", event.GetResourceId().GetDeviceId(), event.GetResourceId().GetHref())
	}
	defer c.blockSignOff.Release(1)
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
	r := &commands.ConfirmDeviceMetadataUpdateRequest{
		DeviceId:      event.GetDeviceId(),
		CorrelationId: event.GetAuditContext().GetCorrelationId(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: c.RemoteAddr().String(),
			Sequence:     c.coapConn.Sequence(),
		},
		Status: commands.Status_OK,
	}
	if event.GetTwinForceSynchronization() {
		r.Confirm = &commands.ConfirmDeviceMetadataUpdateRequest_TwinForceSynchronization{
			TwinForceSynchronization: true,
		}
	} else {
		r.Confirm = &commands.ConfirmDeviceMetadataUpdateRequest_TwinEnabled{
			TwinEnabled: event.GetTwinEnabled(),
		}
	}
	_, err := c.server.raClient.ConfirmDeviceMetadataUpdate(ctx, r)
	return err
}

func (c *session) UpdateDeviceMetadata(ctx context.Context, event *events.DeviceMetadataUpdatePending) error {
	if !c.blockSignOff.TryAcquire(1) {
		return fmt.Errorf("cannot update device('%v') metadata: signOff processing", event.GetDeviceId())
	}
	defer c.blockSignOff.Release(1)
	setDeviceIDToTracerSpan(ctx, c.deviceID())
	authCtx, err := c.GetAuthorizationContext()
	if err != nil {
		c.Close()
		return fmt.Errorf("cannot update device('%v') metadata: %w", event.GetDeviceId(), err)
	}
	switch event.GetUpdatePending().(type) {
	case *events.DeviceMetadataUpdatePending_TwinEnabled:
	case *events.DeviceMetadataUpdatePending_TwinForceSynchronization:
	default:
		return nil
	}
	sendConfirmCtx := authCtx.ToContext(ctx)

	var errObs error
	var previous bool
	if event.GetTwinEnabled() || event.GetTwinForceSynchronization() {
		// if twin is enabled, we need to first update twin synchronization state to sync out
		// and then synchronization state will be updated by other replaceDeviceObserverWithDeviceTwin
		err = c.confirmDeviceMetadataUpdate(sendConfirmCtx, event)
		previous, errObs = c.replaceDeviceObserverWithDeviceTwin(sendConfirmCtx, event.GetTwinEnabled(), event.GetTwinForceSynchronization())
	} else {
		// if twin is disabled, we to stop observation resources to disable all update twin synchronization state
		previous, errObs = c.replaceDeviceObserverWithDeviceTwin(sendConfirmCtx, event.GetTwinEnabled(), false)
		// and then we need to update twin synchronization state to disabled
		err = c.confirmDeviceMetadataUpdate(sendConfirmCtx, event)
	}
	if errObs != nil {
		c.Close()
		return fmt.Errorf("cannot update device('%v') metadata: %w", event.GetDeviceId(), errObs)
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		_, errObs := c.replaceDeviceObserverWithDeviceTwin(sendConfirmCtx, previous, false)
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
	closeFn, err := c.server.ownerCache.Subscribe(owner, onEvent)
	if err != nil {
		return err
	}
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	c.private.closeEventSubscriptions = closeFn
	return nil
}

func (c *session) unsubscribeFromDeviceEvents() {
	closeFn := func() {
		// default no-op
	}
	c.private.mutex.Lock()
	if c.private.closeEventSubscriptions != nil {
		closeFn = c.private.closeEventSubscriptions
		c.private.closeEventSubscriptions = nil
	}
	c.private.mutex.Unlock()
	closeFn()
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
