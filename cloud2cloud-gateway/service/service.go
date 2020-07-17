package service

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/security/oauth/manager"

	"github.com/go-ocf/cloud/cloud2cloud-gateway/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pbGRPC "github.com/go-ocf/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	kitNetHttp "github.com/go-ocf/kit/net/http"
)

//Server handle HTTP request
type Server struct {
	server  *http.Server
	cfg     Config
	handler *RequestHandler
	ln      net.Listener
	rdConn  *grpc.ClientConn
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
		log.Fatalf("cannot create oauth manager: %w", err)
	}

	rdConn, err := grpc.Dial(config.ResourceDirectoryAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	rdClient := pbGRPC.NewGrpcGatewayClient(rdConn)
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

	requestHandler := NewRequestHandler(rdClient, subMgr, emitEvent)

	server := Server{
		server:  NewHTTP(requestHandler, authInterceptor),
		cfg:     config,
		handler: requestHandler,
		ln:      ln,
		rdConn:  rdConn,
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
	}()
	return s.server.Serve(s.ln)
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
	s.cancel()
	return s.server.Shutdown(context.Background())
}
