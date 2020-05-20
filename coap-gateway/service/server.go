package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/ants"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/coap-gateway/uri"
	notificationRA "github.com/go-ocf/cloud/resource-aggregate/cqrs/notification"
	projectionRA "github.com/go-ocf/cloud/resource-aggregate/cqrs/projection"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRD "github.com/go-ocf/cloud/resource-directory/pb/resource-directory"
	pbRS "github.com/go-ocf/cloud/resource-directory/pb/resource-shadow"
	"github.com/go-ocf/cqrs/eventbus"
	"github.com/go-ocf/cqrs/eventstore"
	"github.com/go-ocf/go-coap/v2/blockwise"
	"github.com/go-ocf/go-coap/v2/keepalive"
	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/go-coap/v2/net"
	"github.com/go-ocf/go-coap/v2/tcp"
	"github.com/go-ocf/go-coap/v2/tcp/message/pool"
	kitNetCoap "github.com/go-ocf/kit/net/coap"

	"github.com/go-ocf/kit/log"
	cache "github.com/patrickmn/go-cache"
)

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
	ConnectionsHeartBeat            time.Duration
	BlockWiseTransfer               bool
	BlockWiseTransferSZX            blockwise.SZX

	raClient pbRA.ResourceAggregateClient
	asClient pbAS.AuthorizationServiceClient
	rsClient pbRS.ResourceShadowClient
	rdClient pbRD.ResourceDirectoryClient

	clientContainer               *ClientContainer
	clientContainerByDeviceID     *clientContainerByDeviceID
	updateNotificationContainer   *notificationRA.UpdateNotificationContainer
	retrieveNotificationContainer *notificationRA.RetrieveNotificationContainer
	observeResourceContainer      *observeResourceContainer
	goroutinesPool                *ants.Pool
	oicPingCache                  *cache.Cache

	projection      *projectionRA.Projection
	coapServer      *tcp.Server
	listener        tcp.Listener
	authInterceptor kitNetCoap.Interceptor
	wgDone          *sync.WaitGroup
}

type DialCertManager = interface {
	GetClientTLSConfig() *tls.Config
}

type ListenCertManager = interface {
	GetServerTLSConfig() *tls.Config
}

//NewServer setup coap gateway
func New(config Config, dialCertManager DialCertManager, listenCertManager ListenCertManager, authInterceptor kitNetCoap.Interceptor, store eventstore.EventStore, subscriber eventbus.Subscriber, pool *ants.Pool) *Server {
	oicPingCache := cache.New(cache.NoExpiration, time.Minute)
	oicPingCache.OnEvicted(pingOnEvicted)

	dialTLSConfig := dialCertManager.GetClientTLSConfig()

	raConn, err := grpc.Dial(config.ResourceAggregateAddr, grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	raClient := pbRA.NewResourceAggregateClient(raConn)

	asConn, err := grpc.Dial(config.AuthServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	asClient := pbAS.NewAuthorizationServiceClient(asConn)

	rdConn, err := grpc.Dial(config.ResourceDirectoryAddr, grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)))
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
	rdClient := pbRD.NewResourceDirectoryClient(rdConn)
	rsClient := pbRS.NewResourceShadowClient(rdConn)

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

	s := Server{
		FQDN:                            config.FQDN,
		ExternalPort:                    config.ExternalPort,
		Addr:                            config.Addr,
		RequestTimeout:                  config.RequestTimeout,
		DisableTCPSignalMessageCSM:      config.DisableTCPSignalMessageCSM,
		DisablePeerTCPSignalMessageCSMs: config.DisablePeerTCPSignalMessageCSMs,
		SendErrorTextInResponse:         config.SendErrorTextInResponse,
		ConnectionsHeartBeat:            config.ConnectionsHeartBeat,
		BlockWiseTransfer:               !config.DisableBlockWiseTransfer,
		BlockWiseTransferSZX:            blockWiseTransferSZX,

		Keepalive:     keepAlive,
		IsTLSListener: isTLSListener,
		raClient:      raClient,
		asClient:      asClient,
		rsClient:      rsClient,
		rdClient:      rdClient,

		clientContainer:               &ClientContainer{sessions: make(map[string]*Client)},
		clientContainerByDeviceID:     NewClientContainerByDeviceId(),
		updateNotificationContainer:   notificationRA.NewUpdateNotificationContainer(),
		retrieveNotificationContainer: notificationRA.NewRetrieveNotificationContainer(),
		observeResourceContainer:      NewObserveResourceContainer(),
		goroutinesPool:                pool,
		oicPingCache:                  oicPingCache,
		listener:                      listener,
		authInterceptor:               authInterceptor,
		wgDone:                        new(sync.WaitGroup),
	}

	projection, err := projectionRA.NewProjection(context.Background(), fmt.Sprintf("%v:%v", config.FQDN, config.ExternalPort), store, subscriber, newResourceCtx(&s))
	if err != nil {
		log.Fatalf("cannot create projection for server: %v", err)
	}
	s.projection = projection
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
	client := server.clientContainer.Find(s.Client().RemoteAddr().String())

	switch req.Code {
	case coapCodes.POST, coapCodes.DELETE, coapCodes.PUT, coapCodes.GET:
		if client == nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle command: client not found"), client, coapCodes.InternalServerError, req.Token)
			return
		}
		fnc(s, req, client)
	case coapCodes.Empty:
		if client == nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle command: client not found"), client, coapCodes.InternalServerError, req.Token)
			return
		}
		clientResetHandler(s, req, client)
	case coapCodes.Content:
		// Unregistered observer at a peer send us a notification - inform the peer to remove it
		sendResponse(client, coapCodes.Empty, req.Token, message.TextPlain, nil)
	default:
		deviceID := getDeviceID(client)
		log.Errorf("DeviceId: %v: received invalid code: CoapCode(%v)", deviceID, req.Code)
	}
}

func defaultHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	path, _ := req.Options.Path()

	switch {
	case strings.HasPrefix(path, resourceRoute):
		resourceRouteHandler(s, req, client)
	default:
		deviceID := getDeviceID(client)
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: unknown path %v", deviceID, path), client, coapCodes.NotFound, req.Token)
	}
}

func (server *Server) coapConnOnNew(coapConn *tcp.ClientConn) {
	remoteAddr := coapConn.RemoteAddr().String()
	coapConn.AddOnClose(func() {
		if client, ok := server.clientContainer.Pop(remoteAddr); ok {
			client.OnClose()
		}
	})
	server.clientContainer.Add(remoteAddr, newClient(server, coapConn))
}

func (server *Server) loggingMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		client := server.clientContainer.Find(w.Client().RemoteAddr().String())
		tmp := pool.AcquireMessage(r.Context)
		decodeMsgToDebug(client, r, "RECEIVED-COMMAND")
		next.ServeCOAP(w, r)
	})
}

func (server *Server) authMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		client := server.clientContainer.Find(w.Client().RemoteAddr().String())
		if client == nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle request: client not found"), client, coapCodes.InternalServerError, r.Token)
			return
		}

		ctx := kitNetCoap.CtxWithToken(r.Context, client.loadAuthorizationContext().AccessToken)
		path, _ := r.Options.Path()
		_, err := server.authInterceptor(ctx, r.Code, "/"+path)
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle request to path '%v': %v", path, err), client, coapCodes.Unauthorized, r.Token)
			client.Close()
			return
		}
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
	server.coapServer = tcp.NewServer(opts...)
}

func (server *Server) tlsEnabled() bool {
	return server.IsTLSListener
}

// Serve starts a coapgateway on the configured address in *Server.
func (server *Server) Serve() error {
	server.wgDone.Add(1)
	defer server.wgDone.Done()
	return server.coapServer.Serve(server.listener)
}

// Shutdown turn off server.
func (server *Server) Shutdown() error {
	defer server.wgDone.Wait()
	server.coapServer.Stop()
	return server.listener.Close()
}
