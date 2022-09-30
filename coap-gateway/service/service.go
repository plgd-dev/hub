package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	coapDtlsServer "github.com/plgd-dev/go-coap/v3/dtls/server"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/message/status"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/net/blockwise"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/go-coap/v3/pkg/runner/periodic"
	coapTcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	coapTcpServer "github.com/plgd-dev/go-coap/v3/tcp/server"
	coapUdpClient "github.com/plgd-dev/go-coap/v3/udp/client"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	pbGRPC "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	idClient "github.com/plgd-dev/hub/v2/identity-store/client"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry/otelcoap"
	certManagerServer "github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	"github.com/plgd-dev/hub/v2/pkg/service"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/queue"
	raClient "github.com/plgd-dev/hub/v2/resource-aggregate/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	otelCodes "go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

var authCtxKey = "AuthCtx"

type CoapServer interface {
	Serve() error
	Stop()
}

type tcpServer struct {
	coapServer *coapTcpServer.Server
	listener   coapTcpServer.Listener
}

func (s *tcpServer) Serve() error {
	return s.coapServer.Serve(s.listener)
}

func (s *tcpServer) Stop() {
	s.coapServer.Stop()
}

type dtlsServer struct {
	coapServer *coapDtlsServer.Server
	listener   coapDtlsServer.Listener
}

func (s *dtlsServer) Serve() error {
	return s.coapServer.Serve(s.listener)
}

func (s *dtlsServer) Stop() {
	s.coapServer.Stop()
}

// Service is a configuration of coap-gateway
type Service struct {
	config                Config
	blockWiseTransferSZX  blockwise.SZX
	natsClient            *natsClient.Client
	raClient              *raClient.Client
	isClient              pbIS.IdentityStoreClient
	rdClient              pbGRPC.GrpcGatewayClient
	expirationClientCache *cache.Cache[string, *session]
	coapServer            CoapServer
	authInterceptor       Interceptor
	ctx                   context.Context
	cancel                context.CancelFunc
	taskQueue             *queue.Queue
	devicesStatusUpdater  *devicesStatusUpdater
	resourceSubscriber    *subscriber.Subscriber
	providers             map[string]*oauth2.PlgdProvider
	jwtValidator          *jwt.Validator
	sigs                  chan os.Signal
	ownerCache            *idClient.OwnerCache
	subscriptionsCache    *subscription.SubscriptionsCache
	messagePool           *pool.Pool
	logger                log.Logger
	tracerProvider        trace.TracerProvider
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

func newResourceDirectoryClient(config GrpcServerConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (pbGRPC.GrpcGatewayClient, func(), error) {
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
	rdClient := pbGRPC.NewGrpcGatewayClient(rdConn.GRPC())
	return rdClient, closeRdConn, nil
}

func newTCPListener(config COAPConfig, fileWatcher *fsnotify.Watcher, logger log.Logger) (coapTcpServer.Listener, func(), error) {
	if !config.TLS.Enabled {
		listener, err := net.NewTCPListener("tcp", config.Addr)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot create tcp listener: %w", err)
		}
		closeListener := func() {
			if err := listener.Close(); err != nil {
				logger.Errorf("failed to close tcp listener: %w", err)
			}
		}
		return listener, closeListener, nil
	}

	var closeListener fn.FuncList
	coapsTLS, err := certManagerServer.New(config.TLS.Embedded, fileWatcher, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create tls cert manager: %w", err)
	}
	closeListener.AddFunc(coapsTLS.Close)
	tlsCfgForClient := coapsTLS.GetTLSConfig()
	tlsCfg := MakeGetConfigForClient(tlsCfgForClient)
	listener, err := net.NewTLSListener("tcp", config.Addr, &tlsCfg)
	if err != nil {
		closeListener.Execute()
		return nil, nil, fmt.Errorf("cannot create tcp-tls listener: %w", err)
	}
	closeListener.AddFunc(func() {
		if err := listener.Close(); err != nil {
			logger.Errorf("failed to close tcp-tls listener: %w", err)
		}
	})
	return listener, closeListener.ToFunction(), nil
}

func newDTLSListener(config COAPConfig, fileWatcher *fsnotify.Watcher, logger log.Logger) (coapDtlsServer.Listener, func(), error) {
	var closeListener fn.FuncList
	coapsTLS, err := certManagerServer.New(config.TLS.Embedded, fileWatcher, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create tls cert manager: %w", err)
	}
	closeListener.AddFunc(coapsTLS.Close)
	tlsCfgForClient := coapsTLS.GetTLSConfig()
	tlsCfg := MakeGetConfigForClient(tlsCfgForClient)

	dtlsCfg := dtls.Config{
		GetCertificate: func(chi *dtls.ClientHelloInfo) (*tls.Certificate, error) {
			return tlsCfg.GetCertificate(&tls.ClientHelloInfo{ServerName: chi.ServerName})
		},
		ClientCAs:             tlsCfg.ClientCAs,
		VerifyPeerCertificate: tlsCfg.VerifyPeerCertificate,
		LoggerFactory:         logger.DTLSLoggerFactory(),
		CipherSuites:          []dtls.CipherSuiteID{dtls.TLS_ECDHE_ECDSA_WITH_AES_128_CCM, dtls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, dtls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		ConnectContextMaker: func() (context.Context, func()) {
			return context.WithTimeout(context.Background(), time.Second*4)
		},
	}
	listener, err := net.NewDTLSListener("udp", config.Addr, &dtlsCfg)
	if err != nil {
		closeListener.Execute()
		return nil, nil, fmt.Errorf("cannot create dtls listener: %w", err)
	}
	closeListener.AddFunc(func() {
		if err := listener.Close(); err != nil {
			logger.Errorf("failed to close dtls listener: %w", err)
		}
	})
	return listener, closeListener.ToFunction(), nil
}

func blockWiseTransferSZXFromString(s string) (blockwise.SZX, error) {
	switch strings.ToLower(s) {
	case "16":
		return blockwise.SZX16, nil
	case "32":
		return blockwise.SZX32, nil
	case "64":
		return blockwise.SZX64, nil
	case "128":
		return blockwise.SZX128, nil
	case "256":
		return blockwise.SZX256, nil
	case "512":
		return blockwise.SZX512, nil
	case "1024":
		return blockwise.SZX1024, nil
	case "bert":
		return blockwise.SZXBERT, nil
	}
	return blockwise.SZX(0), fmt.Errorf("invalid value %v", s)
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

func newProviders(ctx context.Context, config AuthorizationConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (map[string]*oauth2.PlgdProvider, *oauth2.PlgdProvider, func(), error) {
	var closeProviders fn.FuncList
	var firstProvider *oauth2.PlgdProvider
	providers := make(map[string]*oauth2.PlgdProvider)
	for _, p := range config.Providers {
		provider, err := oauth2.NewPlgdProvider(ctx, p.Config, fileWatcher, logger, tracerProvider, config.OwnerClaim, config.DeviceIDClaim)
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
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector, "coap-gateway", fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}

	tracerProvider := otelClient.GetTracerProvider()

	queue, err := queue.New(config.TaskQueue)
	if err != nil {
		otelClient.Close()
		return nil, fmt.Errorf("cannot create job queue %w", err)
	}

	nats, err := natsClient.New(config.Clients.Eventbus.NATS, fileWatcher, logger)
	if err != nil {
		otelClient.Close()
		queue.Release()
		return nil, fmt.Errorf("cannot create nats client: %w", err)
	}
	nats.AddCloseFunc(otelClient.Close)
	nats.AddCloseFunc(queue.Release)

	resourceSubscriber, err := subscriber.New(nats.GetConn(),
		config.Clients.Eventbus.NATS.PendingLimits,
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

	blockWiseTransferSZX := blockwise.SZX1024
	if config.APIs.COAP.BlockwiseTransfer.Enabled {
		blockWiseTransferSZX, err = blockWiseTransferSZXFromString(config.APIs.COAP.BlockwiseTransfer.SZX)
		if err != nil {
			nats.Close()
			return nil, fmt.Errorf("blockWiseTransferSZX error: %w", err)
		}
	}

	providers, firstProvider, closeProviders, err := newProviders(ctx, config.APIs.COAP.Authorization, fileWatcher, logger, tracerProvider)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create device providers error: %w", err)
	}
	nats.AddCloseFunc(closeProviders)

	if firstProvider == nil {
		nats.Close()
		return nil, fmt.Errorf("device providers are empty")
	}

	keyCache := jwt.NewKeyCache(firstProvider.OpenID.JWKSURL, firstProvider.HTTP())

	jwtValidator := jwt.NewValidator(keyCache)

	ownerCache := idClient.NewOwnerCache(config.APIs.COAP.Authorization.OwnerClaim, config.APIs.COAP.OwnerCacheExpiration, nats.GetConn(), isClient, func(err error) {
		logger.Errorf("ownerCache error: %w", err)
	})
	nats.AddCloseFunc(ownerCache.Close)

	subscriptionsCache := subscription.NewSubscriptionsCache(resourceSubscriber.Conn(), func(err error) {
		logger.Errorf("subscriptionsCache error: %w", err)
	})

	ctx, cancel := context.WithCancel(ctx)

	s := Service{
		config:               config,
		blockWiseTransferSZX: blockWiseTransferSZX,

		natsClient:            nats,
		raClient:              raClient,
		isClient:              isClient,
		rdClient:              rdClient,
		expirationClientCache: newExpirationClientCache(ctx, config.APIs.COAP.OwnerCacheExpiration),
		authInterceptor:       newAuthInterceptor(),
		devicesStatusUpdater:  newDevicesStatusUpdater(ctx, config.Clients.ResourceAggregate.DeviceStatusExpiration, logger),

		sigs: make(chan os.Signal, 1),

		taskQueue:          queue,
		resourceSubscriber: resourceSubscriber,
		providers:          providers,
		jwtValidator:       jwtValidator,

		ctx:    ctx,
		cancel: cancel,

		ownerCache:         ownerCache,
		subscriptionsCache: subscriptionsCache,
		messagePool:        pool.New(uint32(config.APIs.COAP.MessagePoolSize), 1024),
		logger:             logger,
		tracerProvider:     tracerProvider,
	}

	if err := s.setupCoapServer(fileWatcher, logger); err != nil {
		return nil, fmt.Errorf("cannot setup coap server: %w", err)
	}

	return service.New(&s), nil
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
	return fmt.Errorf(errMsg)
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
		err := onUnprocessedRequestError(req.Code())
		client.logRequestResponse(req, nil, err)
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return
	}
	if err != nil {
		resp = client.createErrorResponse(err, req.Token())
	}
	if resp != nil {
		client.WriteMessage(resp)
		defer client.ReleaseMessage(resp)
	}
	client.logRequestResponse(req, resp, err)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
	}
	if resp != nil {
		otelcoap.MessageSentEvent(req.Context(), resp)
		span.SetAttributes(otelcoap.StatusCodeAttr(resp.Code()))
	}
}

func (s *Service) makeCommandTask(req *mux.Message, client *session, fnc func(req *mux.Message, client *session) (*pool.Message, error)) func() {
	path, _ := req.Options().Path()
	ctx, span := otelcoap.Start(req.Context(), path, req.Code().String(), otelcoap.WithTracerProvider(s.tracerProvider), otelcoap.WithSpanOptions(trace.WithSpanKind(trace.SpanKindServer)))
	span.SetAttributes(semconv.NetPeerNameKey.String(client.deviceID()))
	req.SetContext(ctx)
	otelcoap.MessageReceivedEvent(ctx, req.Message)
	otelcoap.SetRequest(ctx, req.Message)
	return func() {
		defer span.End()
		s.processCommandTask(req, client, span, fnc)
	}
}

func executeCommand(s mux.ResponseWriter, req *mux.Message, server *Service, fnc func(req *mux.Message, client *session) (*pool.Message, error)) {
	client, ok := s.Conn().Context().Value(clientKey).(*session)
	if !ok {
		client = newSession(server, s.Conn().(*coapTcpClient.Conn), "", time.Time{})
		if req.Code() == coapCodes.Empty {
			client.logRequestResponse(req, nil, fmt.Errorf("cannot handle command: client not found"))
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

func (s *Service) coapConnOnNew(coapConn mux.Conn) {
	var tlsDeviceID string
	var tlsValidUntil time.Time
	tlsCon, ok := coapConn.NetConn().(*tls.Conn)
	if ok {
		peerCertificates := tlsCon.ConnectionState().PeerCertificates
		if len(peerCertificates) > 0 {
			deviceID, err := coap.GetDeviceIDFromIdentityCertificate(peerCertificates[0])
			if err == nil {
				tlsDeviceID = deviceID
			}
			tlsValidUntil = peerCertificates[0].NotAfter
		}
	}

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
			client = newSession(s, w.Conn().(*coapTcpClient.Conn), "", time.Time{})
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

// setupCoapServer setup coap server
func (s *Service) setupCoapServer(fileWatcher *fsnotify.Watcher, logger log.Logger) error {
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
		return setHandlerError(uri.ResourceDirectory, err)
	}
	if err := m.Handle(uri.SignUp, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		executeCommand(w, r, s, signUpHandler)
	})); err != nil {
		return setHandlerError(uri.SignUp, err)
	}
	if err := m.Handle(uri.SignIn, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		executeCommand(w, r, s, signInHandler)
	})); err != nil {
		return setHandlerError(uri.SignIn, err)
	}
	if err := m.Handle(uri.ResourceDiscovery, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		executeCommand(w, r, s, resourceDiscoveryHandler)
	})); err != nil {
		return setHandlerError(uri.ResourceDiscovery, err)
	}
	if err := m.Handle(uri.RefreshToken, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		executeCommand(w, r, s, refreshTokenHandler)
	})); err != nil {
		return setHandlerError(uri.RefreshToken, err)
	}

	opts := make([]interface {
		coapTcpServer.Option
		coapDtlsServer.Option
	}, 0, 8)
	opts = append(opts, options.WithBlockwise(s.config.APIs.COAP.BlockwiseTransfer.Enabled, s.blockWiseTransferSZX, s.config.APIs.COAP.KeepAlive.Timeout))
	opts = append(opts, options.WithMux(m))
	opts = append(opts, options.WithContext(s.ctx))
	opts = append(opts, options.WithMaxMessageSize(s.config.APIs.COAP.MaxMessageSize))
	opts = append(opts, options.WithErrors(func(e error) {
		s.logger.Errorf("plgd/go-coap: %w", e)
	}))
	opts = append(opts, options.WithGoPool(func(f func()) error {
		// we call directly function in connection-goroutine because
		// pairing request/response cannot be done in taskQueue for a observe resource.
		// - the observe resource creates task which wait for the response and this wait can be infinite
		// if all task goroutines are processing observations and they are waiting for the responses, which
		// will be stored in task queue.  it happens when we use task queue here.
		return s.taskQueue.Submit(f)
	}))
	if s.config.APIs.COAP.TLS.Enabled && s.config.APIs.COAP.TLS.DTLS.Enabled {
		listener, closeListener, err := newDTLSListener(s.config.APIs.COAP, fileWatcher, logger)
		if err != nil {
			return fmt.Errorf("cannot create listener: %w", err)
		}
		s.natsClient.AddCloseFunc(closeListener)
		dtlsOpts := []coapDtlsServer.Option{
			options.WithKeepAlive(2, s.config.APIs.COAP.KeepAlive.Timeout, func(cc *coapUdpClient.Conn) {
				s.onInactivityConnection(cc)
			}),
			options.WithOnNewConn(func(coapConn *coapUdpClient.Conn) {
				s.coapConnOnNew(coapConn)
			}),
			options.WithTransmission(1, s.config.APIs.COAP.KeepAlive.Timeout, 4),
		}
		for _, o := range opts {
			dtlsOpts = append(dtlsOpts, o)
		}
		s.coapServer = &dtlsServer{
			coapServer: coapDtlsServer.New(dtlsOpts...),
			listener:   listener,
		}
	} else {
		listener, closeListener, err := newTCPListener(s.config.APIs.COAP, fileWatcher, logger)
		if err != nil {
			return fmt.Errorf("cannot create listener: %w", err)
		}
		s.natsClient.AddCloseFunc(closeListener)
		tcpOpts := []coapTcpServer.Option{
			options.WithKeepAlive(1, s.config.APIs.COAP.KeepAlive.Timeout, func(cc *coapTcpClient.Conn) {
				s.onInactivityConnection(cc)
			}),
			options.WithOnNewConn(func(coapConn *coapTcpClient.Conn) {
				s.coapConnOnNew(coapConn)
			}),
		}
		for _, o := range opts {
			tcpOpts = append(tcpOpts, o)
		}
		s.coapServer = &tcpServer{
			coapServer: coapTcpServer.New(tcpOpts...),
			listener:   listener,
		}
	}

	return nil
}

func (s *Service) tlsEnabled() bool {
	return s.config.APIs.COAP.TLS.Enabled
}

func (s *Service) Close() error {
	s.coapServer.Stop()
	return nil
}

func (s *Service) Serve() error {
	return s.coapServer.Serve()
}
