package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/plgd-dev/kit/security/oauth/manager"
	kitSync "github.com/plgd-dev/kit/sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

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

	raClient pbRA.ResourceAggregateClient
	asClient pbAS.AuthorizationServiceClient
	rdClient pbGRPC.GrpcGatewayClient

	clients               *kitSync.Map
	oicPingCache          *cache.Cache
	oauthMgr              *manager.Manager
	expirationClientCache *cache.Cache

	coapServer      *tcp.Server
	listener        tcp.Listener
	authInterceptor kitNetCoap.Interceptor
	wgDone          *sync.WaitGroup
	asConn          *grpc.ClientConn
	rdConn          *grpc.ClientConn
	raConn          *grpc.ClientConn
	ctx             context.Context
	cancel          context.CancelFunc
}

type DialCertManager = interface {
	GetClientTLSConfig() *tls.Config
}

type ListenCertManager = interface {
	GetServerTLSConfig() *tls.Config
}

// New creates server.
func New(config Config, dialCertManager DialCertManager, listenCertManager ListenCertManager) *Server {
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
	raClient := pbRA.NewResourceAggregateClient(raConn)

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

	var keepAlive *keepalive.KeepAlive
	if config.KeepaliveEnable {
		keepAlive = keepalive.New(keepalive.WithConfig(keepalive.MakeConfig(config.KeepaliveTimeoutConnection)))
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

		Keepalive:     keepAlive,
		IsTLSListener: isTLSListener,
		raClient:      raClient,
		asClient:      asClient,
		rdClient:      rdClient,
		asConn:        asConn,
		rdConn:        rdConn,
		raConn:        raConn,

		oauthMgr: oauthMgr,

		clients:               kitSync.NewMap(),
		expirationClientCache: expirationClientCache,
		oicPingCache:          oicPingCache,
		listener:              listener,
		authInterceptor:       NewAuthInterceptor(),
		wgDone:                new(sync.WaitGroup),

		ctx:    ctx,
		cancel: cancel,
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
	client, ok := ToClient(server.clients.Load(s.Client().RemoteAddr().String()))
	if !ok {
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

func (server *Server) coapConnOnNew(coapConn *tcp.ClientConn) {
	remoteAddr := coapConn.RemoteAddr().String()
	coapConn.AddOnClose(func() {
		if client, ok := ToClient(server.clients.PullOut(remoteAddr)); ok {
			client.OnClose()
		}
	})
	server.clients.Store(remoteAddr, newClient(server, coapConn))
}

func (server *Server) loggingMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		client, ok := ToClient(server.clients.Load(w.Client().RemoteAddr().String()))
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
		client, ok := ToClient(server.clients.Load(w.Client().RemoteAddr().String()))
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
		r.Context = kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(r.Context, serviceToken.AccessToken), authCtx.UserID)
		next.ServeCOAP(w, r)
	})
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
	server.wgDone.Add(1)
	defer func() {
		defer server.wgDone.Done()
		server.cancel()
		server.oauthMgr.Close()
		server.asConn.Close()
		server.rdConn.Close()
		server.raConn.Close()
		server.listener.Close()
	}()
	return server.coapServer.Serve(server.listener)
}

// Shutdown turn off server.
func (server *Server) Shutdown() error {
	defer server.wgDone.Wait()
	server.coapServer.Stop()
	return nil
}
