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

	cache "github.com/patrickmn/go-cache"
	"github.com/plgd-dev/hub/coap-gateway/uri"
	pbGRPC "github.com/plgd-dev/hub/grpc-gateway/pb"
	idClient "github.com/plgd-dev/hub/identity-store/client"
	pbIS "github.com/plgd-dev/hub/identity-store/pb"
	"github.com/plgd-dev/hub/pkg/fn"
	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/pkg/net/grpc/client"
	certManagerServer "github.com/plgd-dev/hub/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/pkg/security/jwt"
	"github.com/plgd-dev/hub/pkg/security/oauth2"
	"github.com/plgd-dev/hub/pkg/sync/task/queue"
	raClient "github.com/plgd-dev/hub/resource-aggregate/client"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus"
	natsClient "github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/utils"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
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
	tlsDeviceIDCache      *cache.Cache
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
}

func newExpirationClientCache() *cache.Cache {
	expirationClientCache := cache.New(cache.NoExpiration, time.Minute)
	expirationClientCache.OnEvicted(func(deviceID string, c interface{}) {
		if c == nil {
			return
		}
		client := c.(*Client)
		authCtx, err := client.GetAuthorizationContext()
		if err != nil {
			if err2 := client.Close(); err2 != nil {
				log.Errorf("failed to close client connection on token expiration: %w", err2)
			}
			log.Debugf("device %v token has been expired", authCtx.GetDeviceID())
		}
	})
	return expirationClientCache
}

func newResourceAggregateClient(config ResourceAggregateConfig, resourceSubscriber eventbus.Subscriber, logger log.Logger) (*raClient.Client, func(), error) {
	raConn, err := grpcClient.New(config.Connection, logger)
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

func newIdentityStoreClient(config IdentityStoreConfig, logger log.Logger) (pbIS.IdentityStoreClient, func(), error) {
	isConn, err := grpcClient.New(config.Connection, logger)
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

func newResourceDirectoryClient(config GrpcServerConfig, logger log.Logger) (pbGRPC.GrpcGatewayClient, func(), error) {
	rdConn, err := grpcClient.New(config.Connection, logger)
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

func newTCPListener(config COAPConfig, tlsDeviceIDCache *cache.Cache, logger log.Logger) (tcp.Listener, func(), error) {
	if !config.TLS.Enabled {
		listener, err := net.NewTCPListener("tcp", config.Addr)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot create tcp listener: %w", err)
		}
		closeListener := func() {
			if err := listener.Close(); err != nil {
				log.Errorf("failed to close tcp listener: %w", err)
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
	tlsCfg := tls.Config{
		GetConfigForClient: MakeGetConfigForClient(tlsCfgForClient, tlsDeviceIDCache),
	}
	listener, err := net.NewTLSListener("tcp", config.Addr, &tlsCfg)
	if err != nil {
		closeListener.Execute()
		return nil, nil, fmt.Errorf("cannot create tcp-tls listener: %w", err)
	}
	closeListener.AddFunc(func() {
		if err := listener.Close(); err != nil {
			log.Errorf("failed to close tcp-tls listener: %w", err)
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

func getOnInactivityFn() func(cc inactivity.ClientConn) {
	return func(cc inactivity.ClientConn) {
		client, ok := cc.Context().Value(clientKey).(*Client)
		if ok {
			deviceID := getDeviceID(client)
			log.Errorf("DeviceId: %v: keep alive was reached fail limit:: closing connection", deviceID)
		} else {
			log.Errorf("keep alive was reached fail limit:: closing connection")
		}
		if err := cc.Close(); err != nil {
			log.Errorf("failed to close connection: %w", err)
		}
	}
}

func newProviders(ctx context.Context, config AuthorizationConfig, logger log.Logger) (map[string]*oauth2.PlgdProvider, *oauth2.PlgdProvider, func(), error) {
	var closeProviders fn.FuncList
	var firstProvider *oauth2.PlgdProvider
	providers := make(map[string]*oauth2.PlgdProvider)
	for _, p := range config.Providers {
		provider, err := oauth2.NewPlgdProvider(ctx, p.Config, logger)
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
	queue, err := queue.New(config.TaskQueue)
	if err != nil {
		return nil, fmt.Errorf("cannot create job queue %w", err)
	}

	nats, err := natsClient.New(config.Clients.Eventbus.NATS, logger)
	if err != nil {
		queue.Release()
		return nil, fmt.Errorf("cannot create nats client: %w", err)
	}
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

	raClient, closeRaClient, err := newResourceAggregateClient(config.Clients.ResourceAggregate, resourceSubscriber, logger)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create resource-aggregate client: %w", err)
	}
	nats.AddCloseFunc(closeRaClient)

	isClient, closeIsClient, err := newIdentityStoreClient(config.Clients.IdentityStore, logger)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create identity-store client: %w", err)
	}
	nats.AddCloseFunc(closeIsClient)

	rdClient, closeRdClient, err := newResourceDirectoryClient(config.Clients.ResourceDirectory, logger)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create resource-directory client: %w", err)
	}
	nats.AddCloseFunc(closeRdClient)

	tlsDeviceIDCache := cache.New(config.APIs.COAP.KeepAlive.Timeout, config.APIs.COAP.KeepAlive.Timeout/2)
	listener, closeListener, err := newTCPListener(config.APIs.COAP, tlsDeviceIDCache, logger)
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

	providers, firstProvider, closeProviders, err := newProviders(ctx, config.APIs.COAP.Authorization, logger)
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create device providers error: %w", err)
	}
	nats.AddCloseFunc(closeProviders)

	if firstProvider == nil {
		nats.Close()
		return nil, fmt.Errorf("device providers are empty")
	}

	keyCache := jwt.NewKeyCacheWithHttp(firstProvider.OpenID.JWKSURL, firstProvider.HTTPClient.HTTP())

	jwtValidator := jwt.NewValidatorWithKeyCache(keyCache)

	ownerCache := idClient.NewOwnerCache(config.APIs.COAP.Authorization.OwnerClaim, config.APIs.COAP.OwnerCacheExpiration, nats.GetConn(), isClient, func(err error) {
		log.Errorf("ownerCache error: %w", err)
	})
	nats.AddCloseFunc(ownerCache.Close)

	ctx, cancel := context.WithCancel(ctx)

	s := Service{
		config:                config,
		blockWiseTransferSZX:  blockWiseTransferSZX,
		keepaliveOnInactivity: getOnInactivityFn(),

		natsClient:            nats,
		raClient:              raClient,
		isClient:              isClient,
		rdClient:              rdClient,
		expirationClientCache: newExpirationClientCache(),
		tlsDeviceIDCache:      tlsDeviceIDCache,
		listener:              listener,
		authInterceptor:       newAuthInterceptor(),
		devicesStatusUpdater:  newDevicesStatusUpdater(ctx, config.Clients.ResourceAggregate.DeviceStatusExpiration),

		sigs: make(chan os.Signal, 1),

		taskQueue:          queue,
		resourceSubscriber: resourceSubscriber,
		providers:          providers,
		jwtValidator:       jwtValidator,

		ctx:    ctx,
		cancel: cancel,

		ownerCache: ownerCache,
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
			deviceID = fmt.Sprintf("unknown(%v)", client.remoteAddrString())
		}
	}
	return deviceID
}

func validateCommand(s mux.ResponseWriter, req *mux.Message, server *Service, fnc func(req *mux.Message, client *Client)) {
	client, ok := s.Client().Context().Value(clientKey).(*Client)
	if !ok || client == nil {
		client = newClient(server, s.Client().ClientConn().(*tcp.ClientConn), "")
	}
	err := server.taskQueue.Submit(func() {
		switch req.Code {
		case coapCodes.POST, coapCodes.DELETE, coapCodes.PUT, coapCodes.GET:
			fnc(req, client)
		case coapCodes.Empty:
			if !ok {
				client.logAndWriteErrorResponse(fmt.Errorf("cannot handle command: client not found"), coapCodes.InternalServerError, req.Token)
				if err := client.Close(); err != nil {
					log.Errorf("cannot handle command: %w", err)
				}
				return
			}
			clientResetHandler(req, client)
		case coapCodes.Content:
			// Unregistered observer at a peer send us a notification
			deviceID := getDeviceID(client)
			tmp, err := pool.ConvertFrom(req.Message)
			if err != nil {
				log.Errorf("DeviceId: %v: cannot convert dropped notification: %w", deviceID, err)
			} else {
				decodeMsgToDebug(client, tmp, "DROPPED-NOTIFICATION")
			}
		default:
			deviceID := getDeviceID(client)
			log.Errorf("DeviceId: %v: received invalid code: CoapCode(%v)", deviceID, req.Code)
		}
	})
	if err != nil {
		deviceID := getDeviceID(client)
		if err2 := client.Close(); err2 != nil {
			log.Errorf("cannot handle command: %w", err2)
		}
		log.Errorf("DeviceId: %v: cannot handle request %v by task queue: %w", deviceID, req.String(), err)
	}
}

func defaultHandler(req *mux.Message, client *Client) {
	path, _ := req.Options.Path()

	switch {
	case strings.HasPrefix("/"+path, uri.ResourceRoute):
		resourceRouteHandler(req, client)
	default:
		deviceID := getDeviceID(client)
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: unknown path %v", deviceID, path), coapCodes.NotFound, req.Token)
	}
}

const clientKey = "client"

func (server *Service) coapConnOnNew(coapConn *tcp.ClientConn, tlscon *tls.Conn) {
	var tlsDeviceID string
	v, ok := server.tlsDeviceIDCache.Get(coapConn.RemoteAddr().String())
	if ok {
		tlsDeviceID = v.(string)
	}
	client := newClient(server, coapConn, tlsDeviceID)
	coapConn.SetContextValue(clientKey, client)
	coapConn.AddOnClose(func() {
		client.OnClose()
	})
}

func (server *Service) loggingMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		client, ok := w.Client().Context().Value(clientKey).(*Client)
		if !ok {
			client = newClient(server, w.Client().ClientConn().(*tcp.ClientConn), "")
		}
		tmp, err := pool.ConvertFrom(r.Message)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot convert from mux.Message: %w", err), coapCodes.InternalServerError, r.Token)
			return
		}
		decodeMsgToDebug(client, tmp, "RECEIVED-COMMAND")
		next.ServeCOAP(w, r)
	})
}

func (server *Service) authMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		client, ok := w.Client().Context().Value(clientKey).(*Client)
		if !ok {
			client = newClient(server, w.Client().ClientConn().(*tcp.ClientConn), "")
		}
		authCtx, _ := client.GetAuthorizationContext()
		ctx := context.WithValue(r.Context, &authCtxKey, authCtx)
		path, _ := r.Options.Path()
		logErrorAndCloseClient := func(err error, code coapCodes.Code) {
			client.logAndWriteErrorResponse(err, code, r.Token)
			if err := client.Close(); err != nil {
				log.Errorf("coap server error: %w", err)
			}
		}
		ctx, err := server.authInterceptor(ctx, r.Code, "/"+path)
		if err != nil {
			logErrorAndCloseClient(fmt.Errorf("cannot handle request to path '%v': %w", path, err), coapCodes.Unauthorized)
			return
		}
		r.Context = ctx
		next.ServeCOAP(w, r)
	})
}

//setupCoapServer setup coap server
func (server *Service) setupCoapServer() error {
	m := mux.NewRouter()
	m.Use(server.loggingMiddleware, server.authMiddleware)
	m.DefaultHandle(mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, defaultHandler)
	}))
	if err := m.Handle(uri.ResourceDirectory, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, resourceDirectoryHandler)
	})); err != nil {
		return fmt.Errorf("failed to set %v handler: %w", uri.ResourceDirectory, err)
	}
	if err := m.Handle(uri.SignUp, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, signUpHandler)
	})); err != nil {
		return fmt.Errorf("failed to set %v handler: %w", uri.SignUp, err)
	}
	if err := m.Handle(uri.SignIn, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, signInHandler)
	})); err != nil {
		return fmt.Errorf("failed to set %v handler: %w", uri.SignIn, err)
	}
	if err := m.Handle(uri.ResourceDiscovery, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, resourceDiscoveryHandler)
	})); err != nil {
		return fmt.Errorf("failed to set %v handler: %w", uri.ResourceDiscovery, err)
	}
	if err := m.Handle(uri.RefreshToken, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, refreshTokenHandler)
	})); err != nil {
		return fmt.Errorf("failed to set %v handler: %w", uri.RefreshToken, err)
	}

	opts := make([]tcp.ServerOption, 0, 8)
	opts = append(opts, tcp.WithKeepAlive(1, server.config.APIs.COAP.KeepAlive.Timeout, server.keepaliveOnInactivity))
	opts = append(opts, tcp.WithOnNewClientConn(server.coapConnOnNew))
	opts = append(opts, tcp.WithBlockwise(server.config.APIs.COAP.BlockwiseTransfer.Enabled, server.blockWiseTransferSZX, server.config.APIs.COAP.KeepAlive.Timeout))
	opts = append(opts, tcp.WithMux(m))
	opts = append(opts, tcp.WithContext(server.ctx))
	opts = append(opts, tcp.WithHeartBeat(server.config.APIs.COAP.GoroutineSocketHeartbeat))
	opts = append(opts, tcp.WithMaxMessageSize(server.config.APIs.COAP.MaxMessageSize))
	opts = append(opts, tcp.WithErrors(func(e error) {
		log.Errorf("plgd/go-coap: %w", e)
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
	server.coapServer = tcp.NewServer(opts...)
	return nil
}

func (server *Service) tlsEnabled() bool {
	return server.config.APIs.COAP.TLS.Enabled
}

// Serve starts a coapgateway on the configured address in *Service.
func (server *Service) Serve() error {
	return server.serveWithHandlingSignal()
}

func (server *Service) serveWithHandlingSignal() error {
	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	go func(server *Service) {
		defer wg.Done()
		err = server.coapServer.Serve(server.listener)
		server.cancel()
		server.natsClient.Close()
	}(server)

	signal.Notify(server.sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-server.sigs

	server.coapServer.Stop()
	wg.Wait()

	server.tlsDeviceIDCache.Flush()

	return err
}

// Shutdown turn off server.
func (server *Service) Close() error {
	select {
	case server.sigs <- syscall.SIGTERM:
	default:
	}
	return nil
}
