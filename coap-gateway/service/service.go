package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pion/dtls/v3"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema/plgdtime"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/message/status"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/go-coap/v3/pkg/runner/periodic"
	pbCA "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	pbGRPC "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	idClient "github.com/plgd-dev/hub/v2/identity-store/client"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	coapService "github.com/plgd-dev/hub/v2/pkg/net/coap/service"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry/otelcoap"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	"github.com/plgd-dev/hub/v2/pkg/service"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/queue"
	raClient "github.com/plgd-dev/hub/v2/resource-aggregate/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	pbRD "github.com/plgd-dev/hub/v2/resource-directory/pb"
	otelCodes "go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

var authCtxKey = "AuthCtx"

type resourceDirectoryClient struct {
	pbGRPC.GrpcGatewayClient
	pbRD.ResourceDirectoryClient
}

// Service is a configuration of coap-gateway
type Service struct {
	ctx                        context.Context
	instanceID                 uuid.UUID
	tracerProvider             trace.TracerProvider
	logger                     log.Logger
	isClient                   pbIS.IdentityStoreClient
	rdClient                   *resourceDirectoryClient
	certificateAuthorityClient pbCA.CertificateAuthorityClient
	devicesStatusUpdater       *devicesStatusUpdater
	providers                  map[string]*oauth2.PlgdProvider
	expirationClientCache      *cache.Cache[string, *session]
	taskQueue                  *queue.Queue
	natsClient                 *natsClient.Client
	resourceSubscriber         *subscriber.Subscriber
	authInterceptor            Interceptor
	jwtValidator               *jwt.Validator
	sigs                       chan os.Signal
	ownerCache                 *idClient.OwnerCache
	subscriptionsCache         *subscription.SubscriptionsCache
	messagePool                *pool.Pool
	raClient                   *raClient.Client
	config                     Config
}

func setExpirationClientCache(c *cache.Cache[string, *session], deviceID string, client *session, validJWTUntil time.Time) {
	validJWTUntil = client.getSessionExpiration(validJWTUntil)
	c.Delete(deviceID)
	if validJWTUntil.IsZero() {
		return
	}
	c.LoadOrStore(deviceID, cache.NewElement(client, validJWTUntil, func(client *session) {
		if client == nil {
			return
		}
		now := time.Now()
		exp := client.getSessionExpiration(now)
		if now.After(exp) {
			client.Close()
			client.Debugf("certificate has been expired")
			return
		}
		_, err := client.GetAuthorizationContext()
		if err != nil {
			client.Close()
			client.Debugf("token has been expired")
		}
	}))
}

func newExpirationClientCache(ctx context.Context, interval time.Duration) *cache.Cache[string, *session] {
	expirationClientCache := cache.NewCache[string, *session]()
	add := periodic.New(ctx.Done(), interval)
	add(func(now time.Time) bool {
		expirationClientCache.CheckExpirations(now)
		return true
	})
	return expirationClientCache
}

func newResourceAggregateClient(config ResourceAggregateConfig, resourceSubscriber eventbus.Subscriber, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*raClient.Client, func(), error) {
	raConn, err := grpcClient.New(config.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create connection to resource-aggregate: %w", err)
	}
	closeRaConn := func() {
		if err := raConn.Close(); err != nil {
			if kitNetGrpc.IsContextCanceled(err) {
				return
			}
			logger.Errorf("error occurs during close connection to resource-aggregate: %v", err)
		}
	}
	raClient := raClient.New(raConn.GRPC(), resourceSubscriber)
	return raClient, closeRaConn, nil
}

func newIdentityStoreClient(config IdentityStoreConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (pbIS.IdentityStoreClient, func(), error) {
	isConn, err := grpcClient.New(config.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create connection to identity-store: %w", err)
	}
	closeIsConn := func() {
		if err := isConn.Close(); err != nil {
			if kitNetGrpc.IsContextCanceled(err) {
				return
			}
			logger.Errorf("error occurs during close connection to identity-store: %v", err)
		}
	}
	isClient := pbIS.NewIdentityStoreClient(isConn.GRPC())
	return isClient, closeIsConn, nil
}

func newResourceDirectoryClient(config GrpcServerConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*resourceDirectoryClient, func(), error) {
	rdConn, err := grpcClient.New(config.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create connection to resource-directory: %w", err)
	}
	closeRdConn := func() {
		if err := rdConn.Close(); err != nil {
			if kitNetGrpc.IsContextCanceled(err) {
				return
			}
			logger.Errorf("error occurs during close connection to resource-directory: %v", err)
		}
	}

	return &resourceDirectoryClient{
		GrpcGatewayClient:       pbGRPC.NewGrpcGatewayClient(rdConn.GRPC()),
		ResourceDirectoryClient: pbRD.NewResourceDirectoryClient(rdConn.GRPC()),
	}, closeRdConn, nil
}

func newCertificateAuthorityClient(config GrpcServerConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (pbCA.CertificateAuthorityClient, func(), error) {
	caConn, err := grpcClient.New(config.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to certificate-authority: %w", err)
	}
	closeCaConn := func() {
		if err := caConn.Close(); err != nil {
			logger.Errorf("error occurs during close connection to certificate-authority: %w", err)
		}
	}
	certificateAuthorityClient := pbCA.NewCertificateAuthorityClient(caConn.GRPC())
	return certificateAuthorityClient, closeCaConn, nil
}

func (s *Service) onInactivityConnection(cc mux.Conn) {
	client, ok := cc.Context().Value(clientKey).(*session)
	if ok {
		deviceID := getDeviceID(client)
		client.Errorf("DeviceId: %v: keep alive was reached fail limit:: closing connection", deviceID)
	} else {
		s.logger.Errorf("keep alive was reached fail limit:: closing connection")
	}
	if err := cc.Close(); err != nil {
		s.logger.Errorf("failed to close connection: %w", err)
	}
}

func newProviders(ctx context.Context, config AuthorizationConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, validator *jwt.Validator) (map[string]*oauth2.PlgdProvider, *oauth2.PlgdProvider, func(), error) {
	var closeProviders fn.FuncList
	var firstProvider *oauth2.PlgdProvider
	providers := make(map[string]*oauth2.PlgdProvider, 4)
	for _, p := range config.Providers {
		provider, err := oauth2.NewPlgdProvider(ctx, p.Config, fileWatcher, logger, tracerProvider, config.OwnerClaim, config.DeviceIDClaim, validator)
		if err != nil {
			closeProviders.Execute()
			return nil, nil, nil, fmt.Errorf("cannot create device provider: %w", err)
		}
		closeProviders.AddFunc(provider.Close)
		providers[p.Name] = provider
		if firstProvider == nil {
			firstProvider = provider
		}
	}
	return providers, firstProvider, closeProviders.ToFunction(), nil
}

// New creates server.
func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*service.Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector, "coap-gateway", fileWatcher, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	otelClient.AddCloseFunc(cancel)

	tracerProvider := otelClient.GetTracerProvider()

	queue, err := queue.New(config.TaskQueue)
	if err != nil {
		otelClient.Close()
		return nil, fmt.Errorf("cannot create job queue %w", err)
	}

	nats, err := natsClient.New(config.Clients.Eventbus.NATS.Config, fileWatcher, logger)
	if err != nil {
		otelClient.Close()
		queue.Release()
		return nil, fmt.Errorf("cannot create nats client: %w", err)
	}
	nats.AddCloseFunc(otelClient.Close)
	nats.AddCloseFunc(queue.Release)

	resourceSubscriber, err := subscriber.New(nats.GetConn(),
		config.Clients.Eventbus.NATS.PendingLimits, config.Clients.Eventbus.NATS.LeadResourceType.IsEnabled(),
		logger,
		subscriber.WithGoPool(func(f func()) error { return queue.Submit(f) }),
		subscriber.WithUnmarshaler(utils.Unmarshal))
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create eventbus subscriber: %w", err)
	}
	nats.AddCloseFunc(resourceSubscriber.Close)

	raClient, closeRaClient, err := newResourceAggregateClient(config.Clients.ResourceAggregate, resourceSubscriber, fileWatcher, logger, tracerProvider)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create resource-aggregate client: %w", err)
	}
	nats.AddCloseFunc(closeRaClient)

	isClient, closeIsClient, err := newIdentityStoreClient(config.Clients.IdentityStore, fileWatcher, logger, tracerProvider)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create identity-store client: %w", err)
	}
	nats.AddCloseFunc(closeIsClient)

	rdClient, closeRdClient, err := newResourceDirectoryClient(config.Clients.ResourceDirectory, fileWatcher, logger, tracerProvider)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create resource-directory client: %w", err)
	}
	nats.AddCloseFunc(closeRdClient)

	certificateAuthorityClient, closeCertificateAuthorityClient, err := newCertificateAuthorityClient(config.Clients.CertificateAuthority, fileWatcher, logger, tracerProvider)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create certificate-authority client: %w", err)
	}
	nats.AddCloseFunc(closeCertificateAuthorityClient)

	validator, err := validator.New(ctx, config.APIs.COAP.Authorization.Authority, fileWatcher, logger, tracerProvider)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create jwt validator: %w", err)
	}
	nats.AddCloseFunc(validator.Close)

	providers, firstProvider, closeProviders, err := newProviders(ctx, config.APIs.COAP.Authorization, fileWatcher, logger, tracerProvider, validator.GetParser())
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create device providers error: %w", err)
	}
	nats.AddCloseFunc(closeProviders)

	if firstProvider == nil {
		nats.Close()
		return nil, errors.New("device providers are empty")
	}

	ownerCache := idClient.NewOwnerCache(config.APIs.COAP.Authorization.OwnerClaim, config.APIs.COAP.OwnerCacheExpiration, nats.GetConn(), isClient, func(err error) {
		logger.Errorf("ownerCache error: %w", err)
	})
	nats.AddCloseFunc(ownerCache.Close)

	subscriptionsCache := subscription.NewSubscriptionsCache(resourceSubscriber.Conn(), func(err error) {
		logger.Errorf("subscriptionsCache error: %w", err)
	})

	instanceID := uuid.New()
	s := Service{
		config:     config,
		instanceID: instanceID,

		natsClient:                 nats,
		raClient:                   raClient,
		isClient:                   isClient,
		rdClient:                   rdClient,
		certificateAuthorityClient: certificateAuthorityClient,
		expirationClientCache:      newExpirationClientCache(ctx, config.APIs.COAP.OwnerCacheExpiration),
		authInterceptor:            newAuthInterceptor(),
		devicesStatusUpdater:       newDevicesStatusUpdater(ctx, instanceID, logger),

		sigs: make(chan os.Signal, 1),

		taskQueue:          queue,
		resourceSubscriber: resourceSubscriber,
		providers:          providers,
		jwtValidator:       validator.GetParser(),

		ctx: ctx,

		ownerCache:         ownerCache,
		subscriptionsCache: subscriptionsCache,
		messagePool:        pool.New(config.APIs.COAP.MessagePoolSize, 1024),
		logger:             logger,
		tracerProvider:     tracerProvider,
	}

	ss, err := s.createServices(fileWatcher, logger)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create services: %w", err)
	}
	return ss, nil
}

func getDeviceID(client *session) string {
	deviceID := "unknown"
	if client != nil {
		authCtx, _ := client.GetAuthorizationContext()
		deviceID = authCtx.GetDeviceID()
		if deviceID == "" {
			deviceID = fmt.Sprintf("unknown(%v)", client.RemoteAddr().String())
		}
	}
	return deviceID
}

func onUnprocessedRequestError(code coapCodes.Code) error {
	errMsg := "request from client was dropped"
	if code == coapCodes.Content {
		errMsg = "notification from client was dropped"
	}
	return errors.New(errMsg)
}

func wantToCloseClientOnError(req *mux.Message) bool {
	if req == nil {
		return true
	}
	path, err := req.Options().Path()
	if err != nil {
		return true
	}
	switch path {
	case uri.RefreshToken, uri.SignIn, uri.SignUp:
		return true
	}
	return false
}

func (s *Service) processCommandTask(req *mux.Message, client *session, span trace.Span, fnc func(req *mux.Message, client *session) (*pool.Message, error)) {
	var resp *pool.Message
	var err error
	switch req.Code() {
	case coapCodes.Empty:
		resp, err = clientResetHandler(req, client)
	case coapCodes.POST, coapCodes.DELETE, coapCodes.PUT, coapCodes.GET:
		resp, err = fnc(req, client)
		if err != nil && wantToCloseClientOnError(req) {
			defer func() {
				// Since tls.Conn is async, there is no way to flush or wait for the write, so we must use heuristics.
				time.Sleep(time.Millisecond * 10)
				client.Close()
			}()
		}
	default:
		err = onUnprocessedRequestError(req.Code())
		client.logRequestResponse(req, nil, err)
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return
	}
	if err != nil {
		resp = client.createErrorResponse(err, req.Token())
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
	}
	if resp != nil {
		otelcoap.MessageSentEvent(req.Context(), otelcoap.MakeMessage(resp))
		span.SetAttributes(otelcoap.StatusCodeAttr(resp.Code()))
	}
	client.logRequestResponse(req, resp, err)
	if resp != nil {
		// need to be the last action, because body of response could be used by another goroutine for block wise transfer
		client.WriteMessage(resp)
		defer client.ReleaseMessage(resp)
	}
}

func (s *Service) makeCommandTask(req *mux.Message, client *session, fnc func(req *mux.Message, client *session) (*pool.Message, error)) func() {
	path, _ := req.Options().Path()
	ctx, span := otelcoap.Start(req.Context(), path, req.Code().String(), otelcoap.WithTracerProvider(s.tracerProvider), otelcoap.WithSpanOptions(trace.WithSpanKind(trace.SpanKindServer)))
	span.SetAttributes(semconv.NetPeerNameKey.String(client.deviceID()))
	req.SetContext(ctx)
	otelcoap.MessageReceivedEvent(ctx, otelcoap.MakeMessage(req.Message))
	otelcoap.SetRequest(ctx, req.Message)

	x := struct {
		req    *mux.Message
		client *session
		span   trace.Span
		fnc    func(req *mux.Message, client *session) (*pool.Message, error)
		s      *Service
	}{
		req:    req,
		client: client,
		span:   span,
		fnc:    fnc,
		s:      s,
	}

	return func() {
		defer x.span.End()
		x.s.processCommandTask(x.req, x.client, x.span, x.fnc)
	}
}

func executeCommand(s mux.ResponseWriter, req *mux.Message, server *Service, fnc func(req *mux.Message, client *session) (*pool.Message, error)) {
	client, ok := s.Conn().Context().Value(clientKey).(*session)
	if !ok {
		client = newSession(server, s.Conn(), "", time.Time{})
		if req.Code() == coapCodes.Empty {
			client.logRequestResponse(req, nil, errors.New("cannot handle command: client not found"))
			client.Close()
			return
		}
	}
	req.Hijack()
	err := server.taskQueue.Submit(server.makeCommandTask(req, client, fnc))
	if err != nil {
		client.logRequestResponse(req, nil, fmt.Errorf("cannot handle request by task queue: %w", err))
		client.Close()
	}
}

func statusErrorf(code coapCodes.Code, fmt string, args ...interface{}) error {
	msg := pool.NewMessage(context.Background())
	msg.SetCode(code)
	return status.Errorf(msg, fmt, args...)
}

func defaultHandler(req *mux.Message, client *session) (*pool.Message, error) {
	path, _ := req.Options().Path()

	switch {
	case strings.HasPrefix(path, uri.ResourceRoute):
		return resourceRouteHandler(req, client)
	default:
		return nil, statusErrorf(coapCodes.NotFound, "unknown path %v", path)
	}
}

const clientKey = "client"

func getTLSInfo(conn net.Conn, logger log.Logger) (deviceID string, validUntil time.Time) {
	if tlsCon, ok := conn.(*tls.Conn); ok {
		peerCertificates := tlsCon.ConnectionState().PeerCertificates
		if len(peerCertificates) > 0 {
			deviceID, err := coap.GetDeviceIDFromIdentityCertificate(peerCertificates[0])
			if err == nil {
				return deviceID, peerCertificates[0].NotAfter
			}
			logger.Warnf("cannot get deviceID from certificate %v: %v", peerCertificates[0].Subject.CommonName, err)
			return "", peerCertificates[0].NotAfter
		}
		logger.Debugf("cannot get deviceID from certificate: certificate is not set")
		return "", time.Time{}
	}

	dtlsCon, ok := conn.(*dtls.Conn)
	if !ok {
		logger.Debugf("cannot get deviceID from certificate: unsupported connection type")
		return "", time.Time{}
	}
	cs, ok := dtlsCon.ConnectionState()
	if !ok {
		logger.Debugf("cannot get deviceID from certificate: cannot get connection state")
		return "", time.Time{}
	}
	peerCertificates := cs.PeerCertificates
	if len(peerCertificates) > 0 {
		cert, err := x509.ParseCertificate(peerCertificates[0])
		if err != nil {
			logger.Warnf("cannot get deviceID from certificate: %w", err)
			return "", time.Time{}
		}
		deviceID, err := coap.GetDeviceIDFromIdentityCertificate(cert)
		if err == nil {
			return deviceID, cert.NotAfter
		}
		logger.Warnf("cannot get deviceID from certificate %v: %w", cert.Subject.CommonName, err)
		return "", cert.NotAfter
	}
	logger.Debugf("cannot get deviceID from certificate: certificate is not set")
	return "", time.Time{}
}

func (s *Service) coapConnOnNew(coapConn mux.Conn) {
	tlsDeviceID, tlsValidUntil := getTLSInfo(coapConn.NetConn(), s.logger)
	client := newSession(s, coapConn, tlsDeviceID, tlsValidUntil)
	coapConn.SetContextValue(clientKey, client)
	coapConn.AddOnClose(func() {
		client.OnClose()
	})
}

func (s *Service) authMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		t := time.Now()
		client, ok := w.Conn().Context().Value(clientKey).(*session)
		if !ok {
			client = newSession(s, w.Conn(), "", time.Time{})
		}
		authCtx, _ := client.GetAuthorizationContext()
		ctx := context.WithValue(r.Context(), &authCtxKey, authCtx)
		path, _ := r.Options().Path()
		ctx, err := s.authInterceptor(ctx, r.Code(), path)
		if ctx == nil {
			ctx = r.Context()
		}
		r.SetContext(context.WithValue(ctx, &log.StartTimeKey, t))
		if err != nil {
			err = statusErrorf(coapCodes.Unauthorized, "cannot handle request to path '%v': %w", path, err)
			resp := client.createErrorResponse(err, r.Token())
			defer client.ReleaseMessage(resp)
			client.WriteMessage(resp)
			client.logRequestResponse(r, resp, err)
			client.Close()
			return
		}
		next.ServeCOAP(w, r)
	})
}

// createServices setup services for coap-gateway.
func (s *Service) createServices(fileWatcher *fsnotify.Watcher, logger log.Logger) (*service.Service, error) {
	setHandlerError := func(uri string, err error) error {
		return fmt.Errorf("failed to set %v handler: %w", uri, err)
	}
	m := mux.NewRouter()
	m.Use(s.authMiddleware)
	m.DefaultHandle(mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		executeCommand(w, r, s, defaultHandler)
	}))
	if err := m.Handle(uri.ResourceDirectory, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		executeCommand(w, r, s, resourceDirectoryHandler)
	})); err != nil {
		return nil, setHandlerError(uri.ResourceDirectory, err)
	}
	if err := m.Handle(uri.SignUp, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		executeCommand(w, r, s, signUpHandler)
	})); err != nil {
		return nil, setHandlerError(uri.SignUp, err)
	}
	if err := m.Handle(uri.SignIn, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		executeCommand(w, r, s, signInHandler)
	})); err != nil {
		return nil, setHandlerError(uri.SignIn, err)
	}
	if err := m.Handle(uri.ResourceDiscovery, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		executeCommand(w, r, s, resourceDiscoveryHandler)
	})); err != nil {
		return nil, setHandlerError(uri.ResourceDiscovery, err)
	}
	if err := m.Handle(uri.RefreshToken, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		executeCommand(w, r, s, refreshTokenHandler)
	})); err != nil {
		return nil, setHandlerError(uri.RefreshToken, err)
	}
	if err := m.Handle(plgdtime.ResourceURI, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		executeCommand(w, r, s, plgdTimeHandler)
	})); err != nil {
		return nil, setHandlerError(plgdtime.ResourceURI, err)
	}

	services, err := coapService.New(s.ctx, s.config.APIs.COAP.Config, m, fileWatcher, logger,
		coapService.WithOnNewConnection(s.coapConnOnNew),
		coapService.WithOnInactivityConnection(s.onInactivityConnection),
		coapService.WithMessagePool(s.messagePool),
		coapService.WithOverrideTLS(func(cfg *tls.Config) *tls.Config {
			tlsCfg := MakeGetConfigForClient(cfg, s.config.APIs.COAP.InjectedCOAPConfig.TLSConfig.IdentityPropertiesRequired)
			return &tlsCfg
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create coap-gateway service: %w", err)
	}
	serviceHeartbeat, err := newServiceHeartbeat(s.instanceID, s.config.ServiceHeartbeat.TimeToLive, s.raClient, logger, services)
	if err != nil {
		_ = services.Close()
		return nil, fmt.Errorf("cannot create service heartbeat: %w", err)
	}

	services.Add(serviceHeartbeat)
	return services, err
}
