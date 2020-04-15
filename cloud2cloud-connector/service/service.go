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
	"google.golang.org/grpc/credentials"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	projectionRA "github.com/go-ocf/cloud/resource-aggregate/cqrs/projection"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
)

//Server handle HTTP request
type Server struct {
	server  *http.Server
	cfg     Config
	handler *RequestHandler
	ln      net.Listener
}

type loadDeviceSubscriptionsHandler struct {
	resourceProjection *projectionRA.Projection
}

func (h *loadDeviceSubscriptionsHandler) Handle(ctx context.Context, iter connectorStore.SubscriptionIter) error {
	var sub connectorStore.Subscription
	for iter.Next(ctx, &sub) {
		_, err := h.resourceProjection.Register(ctx, sub.DeviceID)
		if err != nil {
			log.Errorf("cannot register device %v subscription to resource projection: %v", sub.DeviceID, err)
		}
	}
	return iter.Err()
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
	listenTLSConfig.ClientAuth = tls.NoClientCert

	ln, err := tls.Listen("tcp", config.Addr, listenTLSConfig)
	if err != nil {
		log.Fatalf("cannot listen and serve: %v", err)
	}

	raConn, err := grpc.Dial(config.ResourceAggregateAddr, grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	raClient := pbRA.NewResourceAggregateClient(raConn)

	authConn, err := grpc.Dial(config.AuthServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	authClient := pbAS.NewAuthorizationServiceClient(authConn)

	ctx := context.Background()

	resourceProjection, err := projectionRA.NewProjection(ctx, config.FQDN, resourceEventStore, resourceSubscriber, newResourceCtx(store, raClient))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	// load resource subscriptions
	h := loadDeviceSubscriptionsHandler{
		resourceProjection: resourceProjection,
	}
	err = store.LoadSubscriptions(ctx, []connectorStore.SubscriptionQuery{
		{
			Type: connectorStore.Type_Device,
		},
	}, &h)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	_, err = url.Parse(config.OAuthCallback)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	requestHandler := NewRequestHandler(config.OriginCloud, config.OAuthCallback, NewSubscriptionManager(config.EventsURL, authClient, raClient, store, resourceProjection), authClient, raClient, resourceProjection, store)

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
