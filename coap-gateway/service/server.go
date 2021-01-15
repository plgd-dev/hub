package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/plgd-dev/kit/security/certManager"
	"github.com/plgd-dev/kit/security/oauth/manager"

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
	kitNetCoap "github.com/plgd-dev/kit/net/coap"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"

	cache "github.com/patrickmn/go-cache"
	"github.com/plgd-dev/kit/log"
)

var expiredKey = "Expired"

//Server a configuration of coapgateway
type Server struct {
	FQDN                            string // fully qualified domain name of GW
	ExternalPort                    uint16 // used to construct oic/res response
	Addr                            string // Address to listen on, ":COAP" if empty.
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
	oauthMgr              *manager.Manager
	expirationClientCache *cache.Cache

	coapServer      *tcp.Server
	listener        tcp.Listener
	authInterceptor kitNetCoap.Interceptor
	asConn          *grpc.ClientConn
	rdConn          *grpc.ClientConn
	raConn          *grpc.ClientConn
	ctx             context.Context
	cancel          context.CancelFunc

	sigs             chan os.Signal
	oauthCertManager certManager.CertManager
	raCertManager    certManager.CertManager
	asCertManager    certManager.CertManager
	rdCertManager    certManager.CertManager
}

type DialCertManager = interface {
	GetClientTLSConfig() *tls.Config
}

type ListenCertManager = interface {
	GetServerTLSConfig() *tls.Config
}

// New creates server.
func New(config ServiceConfig, clients ClientsConfig) *Server {
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

	oauthCertManager, err := certManager.NewCertManager(clients.OAuthProvider.OAuthTLSConfig)
	if err != nil {
		log.Fatalf("cannot create oauth dial cert manager %v", err)
	}

	oauthDialTLSConfig := oauthCertManager.GetClientTLSConfig()
	oauthMgr, err := manager.NewManagerFromConfiguration(clients.OAuthProvider.OAuthConfig, oauthDialTLSConfig)
	if err != nil {
		log.Fatalf("cannot create oauth manager: %v", err)
	}

	raCertManager, err := certManager.NewCertManager(clients.ResourceAggregate.ResourceAggregateClientTLSConfig)
	if err != nil {
		log.Fatalf("cannot create resource-aggregate dial cert manager %v", err)
	}

	raDialTLSConfig := raCertManager.GetClientTLSConfig()
	raConn, err := grpc.Dial(
		clients.ResourceAggregate.ResourceAggregateAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(raDialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	raClient := pbRA.NewResourceAggregateClient(raConn)

	asCertManager, err := certManager.NewCertManager(clients.Authorization.AuthServerClientTLSConfig)
	if err != nil {
		log.Fatalf("cannot create authorization dial cert manager %v", err)
	}

	asDialTLSConfig := asCertManager.GetClientTLSConfig()
	asConn, err := grpc.Dial(
		clients.Authorization.AuthServerAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(asDialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	asClient := pbAS.NewAuthorizationServiceClient(asConn)

	rdCertManager, err := certManager.NewCertManager(clients.ResourceDirectory.ResourceDirectoryClientTLSConfig)
	if err != nil {
		log.Fatalf("cannot create resource-directory dial cert manager %v", err)
	}

	rdDialTLSConfig := rdCertManager.GetClientTLSConfig()
	rdConn, err := grpc.Dial(clients.ResourceDirectory.ResourceDirectoryAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(rdDialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	rdClient := pbGRPC.NewGrpcGatewayClient(rdConn)

	listenCertManager, err := certManager.NewCertManager(config.CoapGW.ServerTLSConfig)
	if err != nil {
		log.Fatalf("cannot create listen cert manager %v", err)
	}
	var listener tcp.Listener
	var isTLSListener bool
	if listenCertManager == nil || reflect.ValueOf(listenCertManager).IsNil() {
		l, err := net.NewTCPListener("tcp", config.CoapGW.Addr)
		if err != nil {
			log.Fatalf("cannot setup tcp for server: %v", err)
		}
		listener = l
	} else {
		tlsConfig := listenCertManager.GetServerTLSConfig()
		l, err := net.NewTLSListener("tcp", config.CoapGW.Addr, tlsConfig)
		if err != nil {
			log.Fatalf("cannot setup tcp-tls for server: %v", err)
		}
		listener = l
		isTLSListener = true
	}

	var keepAlive *keepalive.KeepAlive
	if config.CoapGW.KeepaliveEnable {
		keepAlive = keepalive.New(keepalive.WithConfig(keepalive.MakeConfig(config.CoapGW.KeepaliveTimeoutConnection)))
	}

	var blockWiseTransferSZX blockwise.SZX
	switch strings.ToLower(config.CoapGW.BlockWiseTransferSZX) {
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
		log.Fatalf("invalid value BlockWiseTransferSZX %v", config.CoapGW.BlockWiseTransferSZX)
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := Server{
		FQDN:                            config.CoapGW.FQDN,
		ExternalPort:                    config.CoapGW.ExternalPort,
		Addr:                            config.CoapGW.Addr,
		RequestTimeout:                  config.CoapGW.RequestTimeout,
		DisableTCPSignalMessageCSM:      config.CoapGW.DisableTCPSignalMessageCSM,
		DisablePeerTCPSignalMessageCSMs: config.CoapGW.DisablePeerTCPSignalMessageCSMs,
		SendErrorTextInResponse:         config.CoapGW.SendErrorTextInResponse,
		BlockWiseTransfer:               config.CoapGW.BlockWiseTransferEnable,
		BlockWiseTransferSZX:            blockWiseTransferSZX,
		ReconnectInterval:               config.CoapGW.ReconnectInterval,
		HeartBeat:                       config.CoapGW.HeartBeat,
		MaxMessageSize:                  config.CoapGW.MaxMessageSize,

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

		ctx:    ctx,
		cancel: cancel,

		oauthCertManager: oauthCertManager,
		raCertManager:    raCertManager,
		asCertManager:    asCertManager,
		rdCertManager:    rdCertManager,
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

func validateCommand(s mux.ResponseWriter, req *mux.Message, server *Server, fnc func(s mux.ResponseWriter, req *mux.Message, client *Client)) {
	client, ok := s.Client().Context().Value(clientKey).(*Client)
	if !ok || client == nil {
		client = newClient(server, s.Client().ClientConn().(*tcp.ClientConn))
	}
	switch req.Code {
	case coapCodes.POST, coapCodes.DELETE, coapCodes.PUT, coapCodes.GET:
		fnc(s, req, client)
	case coapCodes.Empty:
		if !ok {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot handle command: client not found"), coapCodes.InternalServerError, req.Token)
			return
		}
		clientResetHandler(s, req, client)
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
}

func defaultHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	path, _ := req.Options.Path()

	switch {
	case strings.HasPrefix("/"+path, uri.ResourceRoute):
		resourceRouteHandler(s, req, client)
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
	}(server)

	signal.Notify(server.sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-server.sigs

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
