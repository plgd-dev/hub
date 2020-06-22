package service

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"

	"google.golang.org/grpc"

	connectorStore "github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"github.com/go-ocf/cqrs/eventbus"
	cqrsEventStore "github.com/go-ocf/cqrs/eventstore"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/security/oauth/manager"
	"google.golang.org/grpc/credentials"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	pbGRPC "github.com/go-ocf/cloud/grpc-gateway/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
)

//Server handle HTTP request
type Server struct {
	server  *http.Server
	cfg     Config
	handler *RequestHandler
	ln      net.Listener
}

type DialCertManager = interface {
	GetClientTLSConfig() *tls.Config
}

type ListenCertManager = interface {
	GetServerTLSConfig() *tls.Config
}

//New create new Server with provided store and bus
func New(config Config, dialCertManager DialCertManager, listenCertManager ListenCertManager, resourceEventStore cqrsEventStore.EventStore, resourceSubscriber eventbus.Subscriber, store connectorStore.Store) *Server {
	dialTLSConfig := dialCertManager.GetClientTLSConfig()
	listenTLSConfig := listenCertManager.GetServerTLSConfig()

	ln, err := tls.Listen("tcp", config.Addr, listenTLSConfig)
	if err != nil {
		log.Fatalf("cannot listen and serve: %v", err)
	}

	oauthMgr, err := manager.NewManagerFromConfiguration(config.OAuth, dialTLSConfig)
	if err != nil {
		log.Fatalf("cannot create oauth manager: %w", err)
	}

	raConn, err := grpc.Dial(config.ResourceAggregateAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	raClient := pbRA.NewResourceAggregateClient(raConn)

	authConn, err := grpc.Dial(config.AuthServerAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	asClient := pbAS.NewAuthorizationServiceClient(authConn)

	rdConn, err := grpc.Dial(config.ResourceDirectoryAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	rdClient := pbGRPC.NewGrpcGatewayClient(rdConn)

	_, err = url.Parse(config.OAuthCallback)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	devicesSubscription := NewDevicesSubscription(rdClient, raClient)
	requestHandler := NewRequestHandler(config.OAuthCallback, NewSubscriptionManager(config.EventsURL, asClient, raClient, store, devicesSubscription), asClient, raClient, store)

	server := Server{
		server:  NewHTTP(requestHandler),
		cfg:     config,
		handler: requestHandler,
		ln:      ln,
	}

	return &server
}

// Serve starts the service's HTTP server and blocks.
func (s *Server) Serve() error {
	return s.server.Serve(s.ln)
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
	return s.server.Shutdown(context.Background())
}
