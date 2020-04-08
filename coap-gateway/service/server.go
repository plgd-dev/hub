package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"reflect"
	"strings"
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
	coapCodes "github.com/go-ocf/go-coap/codes"
	gocoapNet "github.com/go-ocf/go-coap/net"
	kitNetCoap "github.com/go-ocf/kit/net/coap"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/log"
	cache "github.com/patrickmn/go-cache"
)

//Server a configuration of coapgateway
type Server struct {
	FQDN                            string // fully qualified domain name of GW
	ExternalPort                    uint16 // used to construct oic/res response
	Addr                            string // Address to listen on, ":COAP" if empty.
	Net                             string // if "tcp" or "tcp-tls" (COAP over TLS) it will invoke a TCP listener, otherwise an UDP one
	Keepalive                       gocoap.KeepAlive
	DisableTCPSignalMessageCSM      bool
	DisablePeerTCPSignalMessageCSMs bool
	SendErrorTextInResponse         bool
	RequestTimeout                  time.Duration
	ConnectionsHeartBeat            time.Duration

	raClient pbRA.ResourceAggregateClient
	asClient pbAS.AuthorizationServiceClient
	rsClient pbRS.ResourceShadowClient
	rdClient pbRD.ResourceDirectoryClient

	clientContainer               *ClientContainer
	clientContainerByDeviceId     *clientContainerByDeviceId
	updateNotificationContainer   *notificationRA.UpdateNotificationContainer
	retrieveNotificationContainer *notificationRA.RetrieveNotificationContainer
	observeResourceContainer      *observeResourceContainer
	goroutinesPool                *ants.Pool
	oicPingCache                  *cache.Cache

	projection      *projectionRA.Projection
	coapServer      *gocoap.Server
	listener        gocoap.Listener
	authInterceptor kitNetCoap.Interceptor
}

type DialCertManager = interface {
	GetClientTLSConfig() tls.Config
}

type ListenCertManager = interface {
	GetServerTLSConfig() tls.Config
}

//NewServer setup coap gateway
func New(config Config, dialCertManager DialCertManager, listenCertManager ListenCertManager, authInterceptor kitNetCoap.Interceptor, store eventstore.EventStore, subscriber eventbus.Subscriber, pool *ants.Pool) *Server {
	oicPingCache := cache.New(cache.NoExpiration, time.Minute)
	oicPingCache.OnEvicted(pingOnEvicted)

	clientTLSConfig := dialCertManager.GetClientTLSConfig()

	raConn, err := grpc.Dial(config.ResourceAggregateAddr, grpc.WithTransportCredentials(credentials.NewTLS(&clientTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	raClient := pbRA.NewResourceAggregateClient(raConn)

	asConn, err := grpc.Dial(config.AuthServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(&clientTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	asClient := pbAS.NewAuthorizationServiceClient(asConn)

	rdConn, err := grpc.Dial(config.ResourceDirectoryAddr, grpc.WithTransportCredentials(credentials.NewTLS(&clientTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	var listener gocoap.Listener

	if listenCertManager == nil || reflect.ValueOf(listenCertManager).IsNil() {
		l, err := gocoapNet.NewTCPListener("tcp", config.Addr, time.Millisecond*100)
		if err != nil {
			log.Fatalf("cannot setup tcp for server: %v", err)
		}
		listener = l
	} else {
		tlsConfig := listenCertManager.GetServerTLSConfig()
		l, err := gocoapNet.NewTLSListener("tcp", config.Addr, &tlsConfig, time.Millisecond*100)
		if err != nil {
			log.Fatalf("cannot setup tcp-tls for server: %v", err)
		}
		listener = l
	}
	rdClient := pbRD.NewResourceDirectoryClient(rdConn)
	rsClient := pbRS.NewResourceShadowClient(rdConn)

	var keepalive gocoap.KeepAlive
	if config.KeepaliveEnable {
		keepalive, err = gocoap.MakeKeepAlive(config.KeepaliveTimeoutConnection)
		if err != nil {
			log.Fatalf("cannot setup keepalive for server: %v", err)
		}
	}

	s := Server{
		Keepalive:                       keepalive,
		Net:                             config.Net,
		FQDN:                            config.FQDN,
		ExternalPort:                    config.ExternalPort,
		Addr:                            config.Addr,
		RequestTimeout:                  config.RequestTimeout,
		DisableTCPSignalMessageCSM:      config.DisableTCPSignalMessageCSM,
		DisablePeerTCPSignalMessageCSMs: config.DisablePeerTCPSignalMessageCSMs,
		SendErrorTextInResponse:         config.SendErrorTextInResponse,
		ConnectionsHeartBeat:            config.ConnectionsHeartBeat,

		raClient: raClient,
		asClient: asClient,
		rsClient: rsClient,
		rdClient: rdClient,

		clientContainer:               &ClientContainer{sessions: make(map[string]*Client)},
		clientContainerByDeviceId:     NewClientContainerByDeviceId(),
		updateNotificationContainer:   notificationRA.NewUpdateNotificationContainer(),
		retrieveNotificationContainer: notificationRA.NewRetrieveNotificationContainer(),
		observeResourceContainer:      NewObserveResourceContainer(),
		goroutinesPool:                pool,
		oicPingCache:                  oicPingCache,
		listener:                      listener,
		authInterceptor:               authInterceptor,
	}

	projection, err := projectionRA.NewProjection(context.Background(), fmt.Sprintf("%v:%v", config.FQDN, config.ExternalPort), store, subscriber, newResourceCtx(&s))
	if err != nil {
		log.Fatalf("cannot create projection for server: %v", err)
	}
	s.projection = projection
	return &s
}

func getDeviceId(client *Client) string {
	deviceId := "unknown"
	if client != nil {
		deviceId = client.loadAuthorizationContext().DeviceId
		if deviceId == "" {
			deviceId = fmt.Sprintf("unknown(%v)", client.remoteAddrString())
		}
	}
	return deviceId
}

func validateCommand(s gocoap.ResponseWriter, req *gocoap.Request, server *Server, fnc func(s gocoap.ResponseWriter, req *gocoap.Request, client *Client)) {
	client := server.clientContainer.Find(req.Client.RemoteAddr().String())

	switch req.Msg.Code() {
	case coapCodes.POST, coapCodes.DELETE, coapCodes.PUT, coapCodes.GET:
		if client == nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle command: client not found"), s, client, coapCodes.InternalServerError)
			return
		}
		fnc(s, req, client)
	case coapCodes.Empty:
		if client == nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle command: client not found"), s, client, coapCodes.InternalServerError)
			return
		}
		clientResetHandler(s, req, client)
	case coapCodes.Content:
		// Unregistered observer at a peer send us a notification - inform the peer to remove it
		sendResponse(s, client, coapCodes.Empty, gocoap.TextPlain, nil)
	default:
		deviceID := getDeviceId(client)
		log.Errorf("DeviceId: %v: received invalid code: CoapCode(%v)", deviceID, req.Msg.Code())
	}
}

func defaultHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	path := req.Msg.PathString()

	switch {
	case strings.HasPrefix(path, resourceRoute):
		resourceRouteHandler(s, req, client)
	default:
		deviceId := getDeviceId(client)
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: unknown path %v", deviceId, path), s, client, coapCodes.NotFound)
	}
}

func (server *Server) coapConnOnNew(coapConn *gocoap.ClientConn) {
	remoteAddr := coapConn.RemoteAddr().String()
	server.clientContainer.Add(remoteAddr, newClient(server, coapConn))
}

func (server *Server) coapConnOnClose(coapConn *gocoap.ClientConn, err error) {
	if err != nil {
		log.Errorf("coap connection closed with error: %v", err)
	}
	if client, ok := server.clientContainer.Pop(coapConn.RemoteAddr().String()); ok {
		client.OnClose()
	}

}

func (server *Server) logginMiddleware(next func(gocoap.ResponseWriter, *gocoap.Request)) func(gocoap.ResponseWriter, *gocoap.Request) {
	return func(w gocoap.ResponseWriter, req *gocoap.Request) {
		client := server.clientContainer.Find(req.Client.RemoteAddr().String())
		decodeMsgToDebug(client, req.Msg, "RECEIVED-COMMAND")
		next(w, req)
	}
}

func (server *Server) authMiddleware(next func(gocoap.ResponseWriter, *gocoap.Request)) func(gocoap.ResponseWriter, *gocoap.Request) {
	return func(w gocoap.ResponseWriter, req *gocoap.Request) {
		client := server.clientContainer.Find(req.Client.RemoteAddr().String())
		if client == nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle request: client not found"), w, client, coapCodes.InternalServerError)
			return
		}

		ctx := kitNetCoap.CtxWithToken(req.Ctx, client.loadAuthorizationContext().AccessToken)
		_, err := server.authInterceptor(ctx, req.Msg.Code(), "/"+req.Msg.PathString())
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle request to path '%v': %v", req.Msg.PathString(), err), w, client, coapCodes.Unauthorized)
			client.Close()
			return
		}
		next(w, req)
	}
}

//setupCoapServer setup coap server
func (server *Server) setupCoapServer() {
	mux := gocoap.NewServeMux()
	mux.DefaultHandle(gocoap.HandlerFunc(func(s gocoap.ResponseWriter, req *gocoap.Request) {
		validateCommand(s, req, server, defaultHandler)
	}))
	mux.Handle(uri.ResourceDirectory, gocoap.HandlerFunc(func(s gocoap.ResponseWriter, req *gocoap.Request) {
		validateCommand(s, req, server, resourceDirectoryHandler)
	}))
	mux.Handle(uri.SignUp, gocoap.HandlerFunc(func(s gocoap.ResponseWriter, req *gocoap.Request) {
		validateCommand(s, req, server, signUpHandler)
	}))
	mux.Handle(uri.SecureSignUp, gocoap.HandlerFunc(func(s gocoap.ResponseWriter, req *gocoap.Request) {
		validateCommand(s, req, server, signUpHandler)
	}))
	mux.Handle(uri.SignIn, gocoap.HandlerFunc(func(s gocoap.ResponseWriter, req *gocoap.Request) {
		validateCommand(s, req, server, signInHandler)
	}))
	mux.Handle(uri.SecureSignIn, gocoap.HandlerFunc(func(s gocoap.ResponseWriter, req *gocoap.Request) {
		validateCommand(s, req, server, signInHandler)
	}))
	mux.Handle(uri.ResourceDiscovery, gocoap.HandlerFunc(func(s gocoap.ResponseWriter, req *gocoap.Request) {
		validateCommand(s, req, server, resourceDiscoveryHandler)
	}))
	mux.Handle(uri.ResourcePing, gocoap.HandlerFunc(func(s gocoap.ResponseWriter, req *gocoap.Request) {
		validateCommand(s, req, server, resourcePingHandler)
	}))
	mux.Handle(uri.RefreshToken, gocoap.HandlerFunc(func(s gocoap.ResponseWriter, req *gocoap.Request) {
		validateCommand(s, req, server, refreshTokenHandler)
	}))
	mux.Handle(uri.SecureRefreshToken, gocoap.HandlerFunc(func(s gocoap.ResponseWriter, req *gocoap.Request) {
		validateCommand(s, req, server, refreshTokenHandler)
	}))

	server.coapServer = &gocoap.Server{
		Net:                             server.Net,
		Addr:                            server.Addr,
		DisableTCPSignalMessageCSM:      server.DisableTCPSignalMessageCSM,
		DisablePeerTCPSignalMessageCSMs: server.DisablePeerTCPSignalMessageCSMs,
		KeepAlive:                       server.Keepalive,
		Handler:                         gocoap.HandlerFunc(server.logginMiddleware(server.authMiddleware(mux.ServeCOAP))),
		NotifySessionNewFunc:            server.coapConnOnNew,
		NotifySessionEndFunc:            server.coapConnOnClose,
		HeartBeat:                       server.ConnectionsHeartBeat,
	}
}

func (server *Server) tlsEnabled() bool {
	return strings.HasSuffix(server.Net, "-tls")
}

// Serve starts a coapgateway on the configured address in *Server.
func (server *Server) Serve() error {
	server.setupCoapServer()
	server.coapServer.Listener = server.listener
	return server.coapServer.ActivateAndServe()
}

// Shutdown turn off server.
func (server *Server) Shutdown() error {
	err := server.coapServer.Shutdown()
	server.listener.Close()
	return err
}
