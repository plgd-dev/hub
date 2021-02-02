package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/plgd-dev/kit/task/queue"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/coap-gateway/uri"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/keepalive"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/kit/log"
	kitNetCoap "github.com/plgd-dev/kit/net/coap"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	oAuthClient "github.com/plgd-dev/kit/security/oauth/service/client"

	cache "github.com/patrickmn/go-cache"
)

var expiredKey = "Expired"

//Server a configuration of coapgateway
type Server struct {
	Addr                            string // Address to listen on, ":COAP" if empty.
	ExternalAddress                 string // used to construct oic/res response
	IsTLSListener                   bool
	Keepalive                       *keepalive.KeepAlive
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

	raClient pbRA.ResourceAggregateClient
	asClient pbAS.AuthorizationServiceClient
	rdClient pbGRPC.GrpcGatewayClient

	oicPingCache          *cache.Cache
	oauthMgr              *oAuthClient.Manager
	expirationClientCache *cache.Cache

	coapServer      *tcp.Server
	listener        tcp.Listener
	authInterceptor kitNetCoap.Interceptor
	asConn          *grpc.ClientConn
	rdConn          *grpc.ClientConn
	raConn          *grpc.ClientConn
	ctx             context.Context
	cancel          context.CancelFunc

	taskQueue       *queue.TaskQueue
	sigs            chan os.Signal

	oauthCertManager *client.CertManager
	raCertManager    *client.CertManager
	asCertManager    *client.CertManager
	rdCertManager    *client.CertManager
	listenCertManager *server.CertManager
}

type DialCertManager = interface {
	GetClientTLSConfig() *tls.Config
}

type ListenCertManager = interface {
	GetServerTLSConfig() *tls.Config
}

// New creates server.
func New(logger *zap.Logger, service APIsConfig, clients ClientsConfig) *Server {
	p, err := queue.New(service.CoapGW.NumTaskWorkers, service.CoapGW.LimitTasks)
	if err != nil {
		log.Fatalf("cannot job queue %v", err)
	}
	oicPingCache := cache.New(cache.NoExpiration, time.Minute)
	oicPingCache.OnEvicted(pingOnEvicted)

	expirationClientCache := cache.New(cache.NoExpiration, time.Minute)
	expirationClientCache.OnEvicted(func(deviceID string, c interface{}) {
		client := c.(*Client)
		authCtx := client.loadAuthorizationContext()
		if isExpired(authCtx.Expire) {
			client.Close()
			log.Debugf("device %v token has ben expired", authCtx.GetDeviceId())
		}
	})

	oauthCertManager, err := client.New(clients.OAuthProvider.OAuthTLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create oauth dial cert manager %v", err)
	}

	oauthDialTLSConfig := oauthCertManager.GetTLSConfig()
	oauthMgr, err := oAuthClient.NewManagerFromConfiguration(clients.OAuthProvider.OAuthConfig, oauthDialTLSConfig)
	if err != nil {
		log.Fatalf("cannot create oauth manager: %v", err)
	}

	raCertManager, err := client.New(clients.ResourceAggregate.ResourceAggregateTLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create resource-aggregate dial cert manager %v", err)
	}

	raDialTLSConfig := raCertManager.GetTLSConfig()
	raConn, err := grpc.Dial(
		clients.ResourceAggregate.ResourceAggregateAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(raDialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	raClient := pbRA.NewResourceAggregateClient(raConn)

	asCertManager, err := client.New(clients.Authorization.AuthServerTLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create authorization dial cert manager %v", err)
	}

	asDialTLSConfig := asCertManager.GetTLSConfig()
	asConn, err := grpc.Dial(
		clients.Authorization.AuthServerAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(asDialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	asClient := pbAS.NewAuthorizationServiceClient(asConn)

	rdCertManager, err := client.New(clients.ResourceDirectory.ResourceDirectoryTLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create resource-directory dial cert manager %v", err)
	}

	rdDialTLSConfig := rdCertManager.GetTLSConfig()
	rdConn, err := grpc.Dial(clients.ResourceDirectory.ResourceDirectoryAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(rdDialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	rdClient := pbGRPC.NewGrpcGatewayClient(rdConn)

	listenCertManager, err := server.New(service.CoapGW.ServerTLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create listen cert manager %v", err)
	}
	var listener tcp.Listener
	var isTLSListener bool
	if listenCertManager == nil || reflect.ValueOf(listenCertManager).IsNil() {
		l, err := net.NewTCPListener("tcp", service.CoapGW.Addr)
		if err != nil {
			log.Fatalf("cannot setup tcp for server: %v", err)
		}
		listener = l
	} else {
		tlsConfig := listenCertManager.GetTLSConfig()
		l, err := net.NewTLSListener("tcp", service.CoapGW.Addr, tlsConfig)
		if err != nil {
			log.Fatalf("cannot setup tcp-tls for server: %v", err)
		}
		listener = l
		isTLSListener = true
	}

	var keepAlive *keepalive.KeepAlive
	if service.CoapGW.Capabilities.KeepaliveEnable {
		keepAlive = keepalive.New(keepalive.WithConfig(keepalive.MakeConfig(service.CoapGW.Capabilities.KeepaliveTimeoutConnection)))
	}

	var blockWiseTransferSZX blockwise.SZX
	switch strings.ToLower(service.CoapGW.Capabilities.BlockWiseTransferSZX) {
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
		log.Fatalf("invalid value BlockWiseTransferSZX %v", service.CoapGW.Capabilities.BlockWiseTransferSZX)
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := Server{
		Addr:                            service.CoapGW.Addr,
		ExternalAddress:                 service.CoapGW.ExternalAddress,
		RequestTimeout:                  service.CoapGW.RequestTimeout,
		DisableTCPSignalMessageCSM:      service.CoapGW.Capabilities.DisableTCPSignalMessageCSM,
		DisablePeerTCPSignalMessageCSMs: service.CoapGW.Capabilities.DisablePeerTCPSignalMessageCSMs,
		SendErrorTextInResponse:         service.CoapGW.Capabilities.SendErrorTextInResponse,
		BlockWiseTransfer:               service.CoapGW.Capabilities.BlockWiseTransferEnable,
		BlockWiseTransferSZX:            blockWiseTransferSZX,
		ReconnectInterval:               service.CoapGW.ReconnectInterval,
		HeartBeat:                       service.CoapGW.HeartBeat,
		MaxMessageSize:                  service.CoapGW.Capabilities.MaxMessageSize,

		Keepalive:     keepAlive,
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

		sigs: make(chan os.Signal, 1),

		taskQueue: p,

		ctx:    ctx,
		cancel: cancel,

		oauthCertManager: oauthCertManager,
		raCertManager:    raCertManager,
		asCertManager:    asCertManager,
		rdCertManager:    rdCertManager,
		listenCertManager: listenCertManager,
	}

	s.setupCoapServer()

	return &s
}

func getDeviceID(client *Client) string {
	deviceID := "unknown"
	if client != nil {
		deviceID = client.loadAuthorizationContext().DeviceId
		if deviceID == "" {
			deviceID = fmt.Sprintf("unknown(%v)", client.remoteAddrString())
		}
	}
	return deviceID
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

		authCtx := client.loadAuthorizationContext()
		ctx := kitNetCoap.CtxWithToken(r.Context, authCtx.AccessToken)
		ctx = context.WithValue(ctx, &expiredKey, authCtx.Expire)
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
	opts = append(opts, tcp.WithKeepAlive(server.Keepalive))
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
		server.oauthCertManager.Close()
		server.raCertManager.Close()
		server.asCertManager.Close()
		server.rdCertManager.Close()
		server.listenCertManager.Close()
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

	return server.coapServer.Serve(server.listener)
}

// Shutdown turn off server.
func (server *Server) Shutdown() error {
	select {
	case server.sigs <- syscall.SIGTERM:
	default:
	}
	return nil
}
