package service

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/oauth/manager"

	"github.com/plgd-dev/cloud/cloud2cloud-gateway/store"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	raClient "github.com/plgd-dev/cloud/resource-aggregate/client"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	kitNetHttp "github.com/plgd-dev/kit/net/http"
)

//Server handle HTTP request
type Server struct {
	server  *http.Server
	cfg     Config
	handler *RequestHandler
	ln      net.Listener
	rdConn  *grpc.ClientConn
	raConn  *grpc.ClientConn
	cancel  context.CancelFunc
	doneWg  *sync.WaitGroup
}

type DialCertManager = interface {
	GetClientTLSConfig() *tls.Config
}

type ListenCertManager = interface {
	GetServerTLSConfig() *tls.Config
}

//New create new Server with provided store and bus
func New(
	config Config,
	dialCertManager DialCertManager,
	listenCertManager ListenCertManager,
	authInterceptor kitNetHttp.Interceptor,
	subscriptionStore store.Store,
) *Server {
	dialTLSConfig := dialCertManager.GetClientTLSConfig()
	listenTLSConfig := listenCertManager.GetServerTLSConfig()
	listenTLSConfig.ClientAuth = tls.NoClientCert

	ln, err := tls.Listen("tcp", config.Addr, listenTLSConfig)
	if err != nil {
		log.Fatalf("cannot listen and serve: %v", err)
	}

	oauthMgr, err := manager.NewManagerFromConfiguration(config.OAuth, dialTLSConfig)
	if err != nil {
		log.Fatalf("cannot create oauth manager: %v", err)
	}

	rdConn, err := grpc.Dial(config.ResourceDirectoryAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	rdClient := pbGRPC.NewGrpcGatewayClient(rdConn)

	pool, err := ants.NewPool(config.GoRoutinePoolSize)
	if err != nil {
		log.Fatalf("cannot create goroutine pool: %v", err)
	}

	resourceSubscriber, err := nats.NewSubscriber(config.Nats, pool.Submit, func(err error) { log.Errorf("error occurs during receiving event: %v", err) }, nats.WithTLS(dialCertManager.GetClientTLSConfig()))
	if err != nil {
		log.Fatalf("cannot create eventbus subscriber: %v", err)
	}
	raConn, err := grpc.Dial(
		config.ResourceAggregateAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot connect to resource aggregate: %v", err)
	}
	raClient := raClient.New(raConn, resourceSubscriber)

	emitEvent := createEmitEventFunc(dialTLSConfig, config.EmitEventTimeout)

	ctx, cancel := context.WithCancel(context.Background())
	subMgr := NewSubscriptionManager(ctx, subscriptionStore, rdClient, config.ReconnectInterval, emitEvent)
	err = subMgr.LoadSubscriptions()
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		subMgr.Run()
	}()

	requestHandler := NewRequestHandler(rdClient, raClient, subMgr, emitEvent, config.OwnerClaim)

	server := Server{
		server:  NewHTTP(requestHandler, authInterceptor),
		cfg:     config,
		handler: requestHandler,
		ln:      ln,
		rdConn:  rdConn,
		raConn:  raConn,
		cancel:  cancel,
		doneWg:  &wg,
	}

	return &server
}

// Serve starts the service's HTTP server and blocks.
func (s *Server) Serve() error {
	defer func() {
		s.doneWg.Wait()
		s.rdConn.Close()
		s.raConn.Close()
	}()
	return s.server.Serve(s.ln)
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
	s.cancel()
	return s.server.Shutdown(context.Background())
}
