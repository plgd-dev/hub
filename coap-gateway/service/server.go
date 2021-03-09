package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/plgd-dev/kit/security/oauth/manager"
	kitSync "github.com/plgd-dev/kit/sync"

	"github.com/plgd-dev/kit/sync/task/queue"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/coap-gateway/uri"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	kitNetCoap "github.com/plgd-dev/kit/net/coap"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"

	cache "github.com/patrickmn/go-cache"
	"github.com/plgd-dev/kit/log"
)

var authCtxKey = "AuthCtx"

//Server a configuration of coapgateway
type Server struct {
	FQDN                            string // fully qualified domain name of GW
	ExternalPort                    uint16 // used to construct oic/res response
	Addr                            string // Address to listen on, ":COAP" if empty.
	IsTLSListener                   bool
	KeepaliveTimeoutConnection      time.Duration
	KeepaliveOnInactivity           func(cc inactivity.ClientConn)
	DisableTCPSignalMessageCSM      bool
	DisablePeerTCPSignalMessageCSMs bool
	SendErrorTextInResponse         bool
	RequestTimeout                  time.Duration
	BlockWiseTransfer               bool
	BlockWiseTransferSZX            blockwise.SZX
	ReconnectInterval               time.Duration
	HeartBeat                       time.Duration
	MaxMessageSize                  int
	LogMessages                     bool

	raClient service.ResourceAggregateClient
	asClient pbAS.AuthorizationServiceClient
	rdClient pbGRPC.GrpcGatewayClient

	oicPingCache          *cache.Cache
	oauthMgr              *manager.Manager
	expirationClientCache *cache.Cache

	coapServer              *tcp.Server
	listener                tcp.Listener
	authInterceptor         kitNetCoap.Interceptor
	asConn                  *grpc.ClientConn
	rdConn                  *grpc.ClientConn
	raConn                  *grpc.ClientConn
	ctx                     context.Context
	cancel                  context.CancelFunc
	taskQueue               *queue.Queue
	userDeviceSubscriptions *kitSync.Map
	devicesStatusUpdater    *devicesStatusUpdater
	resourceSubscriber      eventbus.Subscriber

	sigs chan os.Signal
}

type DialCertManager = interface {
	GetClientTLSConfig() *tls.Config
}

type ListenCertManager = interface {
	GetServerTLSConfig() *tls.Config
}

// New creates server.
func New(config Config, dialCertManager DialCertManager, listenCertManager ListenCertManager) *Server {
	p, err := queue.New(config.TaskQueue)
	if err != nil {
		log.Fatalf("cannot job queue %v", err)
	}

	resourceSubscriber, err := nats.NewSubscriber(config.Nats, func(v func()) error { return p.Submit(v) }, func(err error) { log.Errorf("coap-gateway: error occurs during receiving event: %v", err) }, nats.WithTLS(dialCertManager.GetClientTLSConfig()))
	if err != nil {
		log.Fatalf("cannot create resource nats subscriber %v", err)
	}

	oicPingCache := cache.New(cache.NoExpiration, time.Minute)
	oicPingCache.OnEvicted(pingOnEvicted)

	expirationClientCache := cache.New(cache.NoExpiration, time.Minute)
	expirationClientCache.OnEvicted(func(deviceID string, c interface{}) {
		if c == nil {
			return
		}
		client := c.(*Client)
		authCtx, err := client.GetAuthorizationContext()
		if err != nil {
			client.Close()
			log.Debugf("device %v token has ben expired", authCtx.GetDeviceID())
		}
	})

	dialTLSConfig := dialCertManager.GetClientTLSConfig()
	oauthMgr, err := manager.NewManagerFromConfiguration(config.OAuth, dialTLSConfig)
	if err != nil {
		log.Fatalf("cannot create oauth manager: %v", err)
	}

	raConn, err := grpc.Dial(
		config.ResourceAggregateAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	raClient := service.NewResourceAggregateClient(raConn)

	asConn, err := grpc.Dial(
		config.AuthServerAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	asClient := pbAS.NewAuthorizationServiceClient(asConn)

	rdConn, err := grpc.Dial(config.ResourceDirectoryAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	var listener tcp.Listener
	var isTLSListener bool
	if listenCertManager == nil || reflect.ValueOf(listenCertManager).IsNil() {
		l, err := net.NewTCPListener("tcp", config.Addr)
		if err != nil {
			log.Fatalf("cannot setup tcp for server: %v", err)
		}
		listener = l
	} else {
		tlsConfig := listenCertManager.GetServerTLSConfig()
		l, err := net.NewTLSListener("tcp", config.Addr, tlsConfig)
		if err != nil {
			log.Fatalf("cannot setup tcp-tls for server: %v", err)
		}
		listener = l
		isTLSListener = true
	}
	rdClient := pbGRPC.NewGrpcGatewayClient(rdConn)

	onInactivity := func(cc inactivity.ClientConn) {}
	if config.KeepaliveEnable {
		onInactivity = func(cc inactivity.ClientConn) {
			cc.Close()
			client, ok := cc.Context().Value(clientKey).(*Client)
			if ok {
				deviceID := getDeviceID(client)
				log.Errorf("DeviceId: %v: keep alive was reached fail limit:: closing connection", deviceID)
			} else {
				log.Errorf("keep alive was reached fail limit:: closing connection")
			}
		}
	}

	var blockWiseTransferSZX blockwise.SZX
	switch strings.ToLower(config.BlockWiseTransferSZX) {
	case "16":
		blockWiseTransferSZX = blockwise.SZX16
	case "32":
		blockWiseTransferSZX = blockwise.SZX32
	case "64":
		blockWiseTransferSZX = blockwise.SZX64
	case "128":
		blockWiseTransferSZX = blockwise.SZX128
	case "256":
		blockWiseTransferSZX = blockwise.SZX256
	case "512":
		blockWiseTransferSZX = blockwise.SZX512
	case "1024":
		blockWiseTransferSZX = blockwise.SZX1024
	case "bert":
		blockWiseTransferSZX = blockwise.SZXBERT
	default:
		log.Fatalf("invalid value BlockWiseTransferSZX %v", config.BlockWiseTransferSZX)
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := Server{
		FQDN:                            config.FQDN,
		ExternalPort:                    config.ExternalPort,
		Addr:                            config.Addr,
		RequestTimeout:                  config.RequestTimeout,
		DisableTCPSignalMessageCSM:      config.DisableTCPSignalMessageCSM,
		DisablePeerTCPSignalMessageCSMs: config.DisablePeerTCPSignalMessageCSMs,
		SendErrorTextInResponse:         config.SendErrorTextInResponse,
		BlockWiseTransfer:               !config.DisableBlockWiseTransfer,
		BlockWiseTransferSZX:            blockWiseTransferSZX,
		ReconnectInterval:               config.ReconnectInterval,
		HeartBeat:                       config.HeartBeat,
		MaxMessageSize:                  config.MaxMessageSize,
		LogMessages:                     config.LogMessages,
		KeepaliveOnInactivity:           onInactivity,

		IsTLSListener: isTLSListener,
		raClient:      raClient,
		asClient:      asClient,
		rdClient:      rdClient,
		asConn:        asConn,
		rdConn:        rdConn,
		raConn:        raConn,

		oauthMgr: oauthMgr,

		expirationClientCache: expirationClientCache,
		oicPingCache:          oicPingCache,
		listener:              listener,
		authInterceptor:       NewAuthInterceptor(),
		devicesStatusUpdater:  NewDevicesStatusUpdater(ctx, config.DeviceStatusExpiration),

		sigs: make(chan os.Signal, 1),

		taskQueue:          p,
		resourceSubscriber: resourceSubscriber,

		ctx:    ctx,
		cancel: cancel,

		userDeviceSubscriptions: kitSync.NewMap(),
	}

	s.setupCoapServer()

	return &s
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

type userDeviceSubscriptionChannel struct {
	counter int
	userID  string
	store   *kitSync.Map

	channel *client.DeviceSubscriptions
	mutex   sync.Mutex
}

func (c *userDeviceSubscriptionChannel) getOrCreate(ctx context.Context, userID string, rdClient pbGRPC.GrpcGatewayClient) (*client.DeviceSubscriptions, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.channel == nil {
		sub, err := client.NewDeviceSubscriptions(kitNetGrpc.CtxWithUserID(ctx, userID), rdClient, func(err error) {
			log.Errorf("userDeviceSubscriptionChannel: %v", err)
		})
		if err == nil {
			c.channel = sub
		}
		return sub, err
	}
	return c.channel, nil
}

func (c *userDeviceSubscriptionChannel) pop() *client.DeviceSubscriptions {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	ch := c.channel
	c.channel = nil
	return ch
}

func (c *userDeviceSubscriptionChannel) cancel() (wait func(), err error) {
	var cancelSubscription bool
	c.store.ReplaceWithFunc(c.userID, func(oldValue interface{}, oldLoaded bool) (newValue interface{}, delete bool) {
		if oldLoaded == true {
			oldValue.(*userDeviceSubscriptionChannel).counter--
			if oldValue.(*userDeviceSubscriptionChannel).counter == 0 {
				cancelSubscription = true
				return nil, true
			}
			return oldValue, false
		}
		return nil, false
	})
	if cancelSubscription {
		ch := c.pop()
		if ch != nil {
			return ch.Cancel()
		}
	}
	return func() {}, nil
}

func (server *Server) subscribeToDevice(ctx context.Context, userID string, deviceID string, handler *deviceSubscriptionHandlers) (func(context.Context) error, error) {
	if userID == "" {
		return nil, fmt.Errorf("subscribeToDevice: invalid userID")
	}
	if deviceID == "" {
		return nil, fmt.Errorf("subscribeToDevice: invalid deviceID")
	}
	channel := &userDeviceSubscriptionChannel{
		counter: 1,
		userID:  userID,
		store:   server.userDeviceSubscriptions,
	}
	oldValue, ok := server.userDeviceSubscriptions.ReplaceWithFunc(userID, func(oldValue interface{}, oldLoaded bool) (newValue interface{}, delete bool) {
		if oldLoaded == true {
			oldValue.(*userDeviceSubscriptionChannel).counter++
			return oldValue, false
		}
		return channel, false
	})
	if ok {
		channel = oldValue.(*userDeviceSubscriptionChannel)
	}
	cancel := func() {
		wait, err1 := channel.cancel()
		if err1 == nil {
			wait()
		} else {
			log.Errorf("subscribeToDevice: cannot cancel channel for user %v device %v: %v", userID, deviceID, err1)
		}
	}
	ch, err := channel.getOrCreate(ctx, userID, server.rdClient)
	if err != nil {
		cancel()
		return nil, err
	}
	sub, err := ch.Subscribe(ctx, deviceID, handler, handler)
	if err != nil {
		cancel()
		return nil, err
	}
	var cancelled uint32
	return func(context.Context) error {
		if !atomic.CompareAndSwapUint32(&cancelled, 0, 1) {
			return nil
		}
		defer cancel()
		return sub.Cancel(ctx)
	}, nil

}

func validateCommand(s mux.ResponseWriter, req *mux.Message, server *Server, fnc func(req *mux.Message, client *Client)) {
	client, ok := s.Client().Context().Value(clientKey).(*Client)
	if !ok || client == nil {
		client = newClient(server, s.Client().ClientConn().(*tcp.ClientConn))
	}
	err := server.taskQueue.Submit(func() {
		switch req.Code {
		case coapCodes.POST, coapCodes.DELETE, coapCodes.PUT, coapCodes.GET:
			fnc(req, client)
		case coapCodes.Empty:
			if !ok {
				client.logAndWriteErrorResponse(fmt.Errorf("cannot handle command: client not found"), coapCodes.InternalServerError, req.Token)
				return
			}
			clientResetHandler(req, client)
		case coapCodes.Content:
			// Unregistered observer at a peer send us a notification
			deviceID := getDeviceID(client)
			tmp, err := pool.ConvertFrom(req.Message)
			if err != nil {
				log.Errorf("DeviceId: %v: cannot convert dropped notification: %v", deviceID, err)
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
		client.Close()
		log.Errorf("DeviceId: %v: cannot handle request %v by task queue: %v", deviceID, req.String(), err)
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

func (server *Server) coapConnOnNew(coapConn *tcp.ClientConn, tlscon *tls.Conn) {
	client := newClient(server, coapConn)
	coapConn.SetContextValue(clientKey, client)
	coapConn.AddOnClose(func() {
		client.OnClose()
	})
}

func (server *Server) loggingMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		client, ok := w.Client().Context().Value(clientKey).(*Client)
		if !ok {
			client = newClient(server, w.Client().ClientConn().(*tcp.ClientConn))
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

func (server *Server) authMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		client, ok := w.Client().Context().Value(clientKey).(*Client)
		if !ok {
			client = newClient(server, w.Client().ClientConn().(*tcp.ClientConn))
		}
		authCtx, _ := client.GetAuthorizationContext()
		ctx := context.WithValue(r.Context, &authCtxKey, authCtx)
		path, _ := r.Options.Path()
		_, err := server.authInterceptor(ctx, r.Code, "/"+path)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot handle request to path '%v': %w", path, err), coapCodes.Unauthorized, r.Token)
			client.Close()
			return
		}
		serviceToken, err := server.oauthMgr.GetToken(r.Context)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot get service token: %w", err), coapCodes.InternalServerError, r.Token)
			client.Close()
			return
		}
		r.Context = kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(r.Context, serviceToken.AccessToken), authCtx.GetUserID())
		next.ServeCOAP(w, r)
	})
}

func (server *Server) ServiceRequestContext(userID string) (context.Context, error) {
	serviceToken, err := server.oauthMgr.GetToken(server.ctx)
	if err != nil {
		return nil, err
	}
	return kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(server.ctx, serviceToken.AccessToken), userID), nil
}

//setupCoapServer setup coap server
func (server *Server) setupCoapServer() {
	m := mux.NewRouter()
	m.Use(server.loggingMiddleware, server.authMiddleware)
	m.DefaultHandle(mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, defaultHandler)
	}))
	m.Handle(uri.ResourceDirectory, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, resourceDirectoryHandler)
	}))
	m.Handle(uri.SignUp, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, signUpHandler)
	}))
	m.Handle(uri.SecureSignUp, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, signUpHandler)
	}))
	m.Handle(uri.SignIn, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, signInHandler)
	}))
	m.Handle(uri.SecureSignIn, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, signInHandler)
	}))
	m.Handle(uri.ResourceDiscovery, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, resourceDiscoveryHandler)
	}))
	m.Handle(uri.ResourcePing, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, resourcePingHandler)
	}))
	m.Handle(uri.RefreshToken, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, refreshTokenHandler)
	}))
	m.Handle(uri.SecureRefreshToken, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, server, refreshTokenHandler)
	}))

	opts := make([]tcp.ServerOption, 0, 8)
	if server.DisableTCPSignalMessageCSM {
		opts = append(opts, tcp.WithDisableTCPSignalMessageCSM())
	}
	if server.DisablePeerTCPSignalMessageCSMs {
		opts = append(opts, tcp.WithDisablePeerTCPSignalMessageCSMs())
	}
	opts = append(opts, tcp.WithKeepAlive(3, server.KeepaliveTimeoutConnection/3, server.KeepaliveOnInactivity))
	opts = append(opts, tcp.WithOnNewClientConn(server.coapConnOnNew))
	opts = append(opts, tcp.WithBlockwise(server.BlockWiseTransfer, server.BlockWiseTransferSZX, server.RequestTimeout))
	opts = append(opts, tcp.WithMux(m))
	opts = append(opts, tcp.WithContext(server.ctx))
	opts = append(opts, tcp.WithHeartBeat(server.HeartBeat))
	opts = append(opts, tcp.WithMaxMessageSize(server.MaxMessageSize))
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
}

func (server *Server) tlsEnabled() bool {
	return server.IsTLSListener
}

// Serve starts a coapgateway on the configured address in *Server.
func (server *Server) Serve() error {
	return server.serveWithHandlingSignal()
}

func (server *Server) serveWithHandlingSignal() error {
	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	go func(server *Server) {
		defer wg.Done()
		err = server.coapServer.Serve(server.listener)
		server.cancel()
		server.oauthMgr.Close()
		server.asConn.Close()
		server.rdConn.Close()
		server.raConn.Close()
		server.listener.Close()
	}(server)

	signal.Notify(server.sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-server.sigs
	server.taskQueue.Release()

	server.coapServer.Stop()
	wg.Wait()

	return err
}

// Shutdown turn off server.
func (server *Server) Shutdown() error {
	select {
	case server.sigs <- syscall.SIGTERM:
	default:
	}
	return nil
}
