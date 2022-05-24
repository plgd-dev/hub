package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/message/status"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v2/pkg/cache"
	"github.com/plgd-dev/go-coap/v2/pkg/runner/periodic"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	pbGRPC "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	idClient "github.com/plgd-dev/hub/v2/identity-store/client"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	certManagerServer "github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/queue"
	raClient "github.com/plgd-dev/hub/v2/resource-aggregate/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

var authCtxKey = "AuthCtx"

// Service is a configuration of coap-gateway
type Service struct {
	config Config

	keepaliveOnInactivity func(cc inactivity.ClientConn)
	blockWiseTransferSZX  blockwise.SZX
	natsClient            *natsClient.Client
	raClient              *raClient.Client
	isClient              pbIS.IdentityStoreClient
	rdClient              pbGRPC.GrpcGatewayClient
	expirationClientCache *cache.Cache
	coapServer            *tcp.Server
	listener              tcp.Listener
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

func setExpirationClientCache(c *cache.Cache, deviceID string, client *Client, validJWTUntil time.Time) {
	validJWTUntil = client.getClientExpiration(validJWTUntil)
	c.Delete(deviceID)
	if validJWTUntil.IsZero() {
		return
	}
	c.LoadOrStore(deviceID, cache.NewElement(client, validJWTUntil, func(cl interface{}) {
		if cl == nil {
			return
		}
		client := cl.(*Client)
		now := time.Now()
		exp := client.getClientExpiration(now)
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

func newExpirationClientCache(ctx context.Context, interval time.Duration) *cache.Cache {
	expirationClientCache := cache.NewCache()
	add := periodic.New(ctx.Done(), interval)
	add(func(now time.Time) bool {
		expirationClientCache.CheckExpirations(now)
		return true
	})
	return expirationClientCache
}

func newResourceAggregateClient(config ResourceAggregateConfig, resourceSubscriber eventbus.Subscriber, logger log.Logger, tracerProvider trace.TracerProvider) (*raClient.Client, func(), error) {
	raConn, err := grpcClient.New(config.Connection, logger, tracerProvider)
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

func newIdentityStoreClient(config IdentityStoreConfig, logger log.Logger, tracerProvider trace.TracerProvider) (pbIS.IdentityStoreClient, func(), error) {
	isConn, err := grpcClient.New(config.Connection, logger, tracerProvider)
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

func newResourceDirectoryClient(config GrpcServerConfig, logger log.Logger, tracerProvider trace.TracerProvider) (pbGRPC.GrpcGatewayClient, func(), error) {
	rdConn, err := grpcClient.New(config.Connection, logger, tracerProvider)
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

func newTCPListener(config COAPConfig, logger log.Logger) (tcp.Listener, func(), error) {
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
	coapsTLS, err := certManagerServer.New(config.TLS.Embedded, logger)
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

func getOnInactivityFn(logger log.Logger) func(cc inactivity.ClientConn) {
	return func(cc inactivity.ClientConn) {
		client, ok := cc.Context().Value(clientKey).(*Client)
		if ok {
			deviceID := getDeviceID(client)
			client.Errorf("DeviceId: %v: keep alive was reached fail limit:: closing connection", deviceID)
		} else {
			logger.Errorf("keep alive was reached fail limit:: closing connection")
		}
		if err := cc.Close(); err != nil {
			logger.Errorf("failed to close connection: %w", err)
		}
	}
}

func newProviders(ctx context.Context, config AuthorizationConfig, logger log.Logger, tracerProvider trace.TracerProvider) (map[string]*oauth2.PlgdProvider, *oauth2.PlgdProvider, func(), error) {
	var closeProviders fn.FuncList
	var firstProvider *oauth2.PlgdProvider
	providers := make(map[string]*oauth2.PlgdProvider)
	for _, p := range config.Providers {
		provider, err := oauth2.NewPlgdProvider(ctx, p.Config, logger, tracerProvider, config.OwnerClaim, config.DeviceIDClaim)
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
func New(ctx context.Context, config Config, logger log.Logger) (*Service, error) {
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector, "coap-gateway", logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}

	tracerProvider := otelClient.GetTracerProvider()

	queue, err := queue.New(config.TaskQueue)
	if err != nil {
		otelClient.Close()
		return nil, fmt.Errorf("cannot create job queue %w", err)
	}

	nats, err := natsClient.New(config.Clients.Eventbus.NATS, logger)
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

	raClient, closeRaClient, err := newResourceAggregateClient(config.Clients.ResourceAggregate, resourceSubscriber, logger, tracerProvider)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create resource-aggregate client: %w", err)
	}
	nats.AddCloseFunc(closeRaClient)

	isClient, closeIsClient, err := newIdentityStoreClient(config.Clients.IdentityStore, logger, tracerProvider)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create identity-store client: %w", err)
	}
	nats.AddCloseFunc(closeIsClient)

	rdClient, closeRdClient, err := newResourceDirectoryClient(config.Clients.ResourceDirectory, logger, tracerProvider)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create resource-directory client: %w", err)
	}
	nats.AddCloseFunc(closeRdClient)

	listener, closeListener, err := newTCPListener(config.APIs.COAP, logger)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create listener: %w", err)
	}
	nats.AddCloseFunc(closeListener)

	blockWiseTransferSZX := blockwise.SZX1024
	if config.APIs.COAP.BlockwiseTransfer.Enabled {
		blockWiseTransferSZX, err = blockWiseTransferSZXFromString(config.APIs.COAP.BlockwiseTransfer.SZX)
		if err != nil {
			nats.Close()
			return nil, fmt.Errorf("blockWiseTransferSZX error: %w", err)
		}
	}

	providers, firstProvider, closeProviders, err := newProviders(ctx, config.APIs.COAP.Authorization, logger, tracerProvider)
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
		config:                config,
		blockWiseTransferSZX:  blockWiseTransferSZX,
		keepaliveOnInactivity: getOnInactivityFn(logger),

		natsClient:            nats,
		raClient:              raClient,
		isClient:              isClient,
		rdClient:              rdClient,
		expirationClientCache: newExpirationClientCache(ctx, config.APIs.COAP.OwnerCacheExpiration),
		listener:              listener,
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

	if err := s.setupCoapServer(); err != nil {
		return nil, fmt.Errorf("cannot setup coap server: %w", err)
	}

	return &s, nil
}

func getDeviceID(client *Client) string {
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
	path, err := req.Options.Path()
	if err != nil {
		return true
	}
	switch path {
	case uri.RefreshToken, uri.SignIn, uri.SignUp:
		return true
	}
	return false
}

func (s *Service) processCommandTask(req *mux.Message, client *Client, span trace.Span, fnc func(req *mux.Message, client *Client) (*pool.Message, error)) {
	var resp *pool.Message
	var err error
	switch req.Code {
	case coapCodes.Empty:
		resp, err = clientResetHandler(req, client)
	case coapCodes.POST, coapCodes.DELETE, coapCodes.PUT, coapCodes.GET:
		resp, err = fnc(req, client)
		if err != nil && wantToCloseClientOnError(req) {
			defer client.Close()
		}
	default:
		err := onUnprocessedRequestError(req.Code)
		client.logRequestResponse(req, nil, err)
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return
	}
	if err != nil {
		resp = client.createErrorResponse(err, req.Token)
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
		messageSent.Event(req.Context, resp)
		span.SetAttributes(statusCodeAttr(resp.Code()))
	}
}

func (s *Service) makeCommandTask(req *mux.Message, client *Client, fnc func(req *mux.Message, client *Client) (*pool.Message, error)) func() {
	tracer := client.server.tracerProvider.Tracer(
		opentelemetry.InstrumentationName,
		trace.WithInstrumentationVersion(opentelemetry.SemVersion()),
	)

	path, _ := req.Options.Path()
	attrs := []attribute.KeyValue{
		semconv.NetPeerNameKey.String(client.deviceID()),
		COAPMethodKey.String(req.Code.String()),
		COAPPathKey.String(path),
	}

	ctx, span := tracer.Start(req.Context, defaultTransportFormatter(path), trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attrs...))
	req.Context = ctx

	tmp, err := client.server.messagePool.ConvertFrom(req.Message)
	if err == nil {
		messageReceived.Event(ctx, tmp)
	}
	return func() {
		defer span.End()
		s.processCommandTask(req, client, span, fnc)
	}
}

func executeCommand(s mux.ResponseWriter, req *mux.Message, server *Service, fnc func(req *mux.Message, client *Client) (*pool.Message, error)) {
	client, ok := s.Client().Context().Value(clientKey).(*Client)
	if !ok {
		client = newClient(server, s.Client().ClientConn().(*tcp.ClientConn), "", time.Time{})
		if req.Code == coapCodes.Empty {
			client.logRequestResponse(req, nil, fmt.Errorf("cannot handle command: client not found"))
			client.Close()
			return
		}
	}
	err := server.taskQueue.Submit(server.makeCommandTask(req, client, fnc))
	if err != nil {
		client.logRequestResponse(req, nil, fmt.Errorf("cannot handle request by task queue: %w", err))
		client.Close()
	}
}

func statusErrorf(code coapCodes.Code, fmt string, args ...interface{}) error {
	return status.Errorf(&message.Message{
		Code: code,
	}, fmt, args...)
}

func defaultHandler(req *mux.Message, client *Client) (*pool.Message, error) {
	path, _ := req.Options.Path()

	switch {
	case strings.HasPrefix(path, uri.ResourceRoute):
		return resourceRouteHandler(req, client)
	default:
		return nil, statusErrorf(coapCodes.NotFound, "unknown path %v", path)
	}
}

const clientKey = "client"

func (s *Service) coapConnOnNew(coapConn *tcp.ClientConn, tlscon *tls.Conn) {
	var tlsDeviceID string
	var tlsValidUntil time.Time
	if tlscon != nil {
		peerCertificates := tlscon.ConnectionState().PeerCertificates
		if len(peerCertificates) > 0 {
			deviceID, err := coap.GetDeviceIDFromIdentityCertificate(peerCertificates[0])
			if err == nil {
				tlsDeviceID = deviceID
			}
			tlsValidUntil = peerCertificates[0].NotAfter
		}
	}

	client := newClient(s, coapConn, tlsDeviceID, tlsValidUntil)
	coapConn.SetContextValue(clientKey, client)
	coapConn.AddOnClose(func() {
		client.OnClose()
	})
}

func (s *Service) authMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		t := time.Now()
		client, ok := w.Client().Context().Value(clientKey).(*Client)
		if !ok {
			client = newClient(s, w.Client().ClientConn().(*tcp.ClientConn), "", time.Time{})
		}
		authCtx, _ := client.GetAuthorizationContext()
		ctx := context.WithValue(r.Context, &authCtxKey, authCtx)
		path, _ := r.Options.Path()
		ctx, err := s.authInterceptor(ctx, r.Code, path)
		if ctx == nil {
			ctx = r.Context
		}
		r.Context = context.WithValue(ctx, &log.StartTimeKey, t)
		if err != nil {
			err = statusErrorf(coapCodes.Unauthorized, "cannot handle request to path '%v': %w", path, err)
			resp := client.createErrorResponse(err, r.Token)
			defer client.ReleaseMessage(resp)
			client.WriteMessage(resp)
			client.logRequestResponse(r, resp, err)
			client.Close()
			return
		}
		next.ServeCOAP(w, r)
	})
}

//setupCoapServer setup coap server
func (s *Service) setupCoapServer() error {
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

	opts := make([]tcp.ServerOption, 0, 8)
	opts = append(opts, tcp.WithKeepAlive(1, s.config.APIs.COAP.KeepAlive.Timeout, s.keepaliveOnInactivity))
	opts = append(opts, tcp.WithOnNewClientConn(s.coapConnOnNew))
	opts = append(opts, tcp.WithBlockwise(s.config.APIs.COAP.BlockwiseTransfer.Enabled, s.blockWiseTransferSZX, s.config.APIs.COAP.KeepAlive.Timeout))
	opts = append(opts, tcp.WithMux(m))
	opts = append(opts, tcp.WithContext(s.ctx))
	opts = append(opts, tcp.WithMaxMessageSize(s.config.APIs.COAP.MaxMessageSize))
	opts = append(opts, tcp.WithErrors(func(e error) {
		s.logger.Errorf("plgd/go-coap: %w", e)
	}))
	opts = append(opts, tcp.WithGoPool(func(f func()) error {
		// we call directly function in connection-goroutine because
		// pairing request/response cannot be done in taskQueue for a observe resource.
		// - the observe resource creates task which wait for the response and this wait can be infinite
		// if all task goroutines are processing observations and they are waiting for the responses, which
		// will be stored in task queue.  it happens when we use task queue here.
		f()
		return nil
	}))
	s.coapServer = tcp.NewServer(opts...)
	return nil
}

func (s *Service) tlsEnabled() bool {
	return s.config.APIs.COAP.TLS.Enabled
}

// Serve starts a coapgateway on the configured address in *Service.
func (s *Service) Serve() error {
	return s.serveWithHandlingSignal()
}

func (s *Service) serveWithHandlingSignal() error {
	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	go func(server *Service) {
		defer wg.Done()
		err = server.coapServer.Serve(server.listener)
		server.cancel()
		server.natsClient.Close()
	}(s)

	signal.Notify(s.sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-s.sigs

	s.coapServer.Stop()
	wg.Wait()

	return err
}

// Close turns off the server.
func (s *Service) Close() error {
	select {
	case s.sigs <- syscall.SIGTERM:
	default:
	}
	return nil
}
