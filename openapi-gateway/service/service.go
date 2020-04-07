package service

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/go-ocf/cqrs/eventbus"
	cqrsEventStore "github.com/go-ocf/cqrs/eventstore"
	"github.com/go-ocf/kit/log"

	oapiStore "github.com/go-ocf/cloud/openapi-connector/store"
	"github.com/go-ocf/cloud/openapi-gateway/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	kitNetHttp "github.com/go-ocf/kit/net/http"
	raCqrs "github.com/go-ocf/cloud/resource-aggregate/cqrs/notification"
	projectionRA "github.com/go-ocf/cloud/resource-aggregate/cqrs/projection"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbDD "github.com/go-ocf/cloud/resource-directory/pb/device-directory"
	pbRD "github.com/go-ocf/cloud/resource-directory/pb/resource-directory"
	pbRS "github.com/go-ocf/cloud/resource-directory/pb/resource-shadow"
)

//Server handle HTTP request
type Server struct {
	server              *http.Server
	cfg                 Config
	handler             *RequestHandler
	ln                  net.Listener
	devicesSubscription *devicesSubscription
}

type ResourceSubscriptionLoader struct {
	projection *projectionRA.Projection
}

func newResourceSubscriptionLoader(projection *projectionRA.Projection) *ResourceSubscriptionLoader {
	return &ResourceSubscriptionLoader{projection: projection}
}

func (l *ResourceSubscriptionLoader) Handle(ctx context.Context, iter store.SubscriptionIter) error {
	var s store.Subscription
	for iter.Next(ctx, &s) {
		_, err := l.projection.Register(ctx, s.DeviceID)
		if err != nil {
			log.Errorf("cannot register to resource projection for resource subscription %v: %v", s.ID, err)
		}
	}
	return iter.Err()
}

type DeviceSubscriptionLoader struct {
	resourceProjection *projectionRA.Projection
}

func newDeviceSubscriptionLoader(resourceProjection *projectionRA.Projection) *DeviceSubscriptionLoader {
	return &DeviceSubscriptionLoader{
		resourceProjection: resourceProjection,
	}
}

func (l *DeviceSubscriptionLoader) Handle(ctx context.Context, iter store.SubscriptionIter) error {
	var s store.Subscription
	for iter.Next(ctx, &s) {
		_, err := l.resourceProjection.Register(ctx, s.DeviceID)
		if err != nil {
			log.Errorf("cannot register to resource projection for device subscription %v: %v", s.ID, err)
		}
	}
	return iter.Err()
}

type DialCertManager = interface {
	GetClientTLSConfig() tls.Config
}

type ListenCertManager = interface {
	GetServerTLSConfig() tls.Config
}

//New create new Server with provided store and bus
func New(
	config Config,
	dialCertManager DialCertManager,
	listenCertManager ListenCertManager,
	authInterceptor kitNetHttp.Interceptor,
	resourceEventStore cqrsEventStore.EventStore,
	resourceSubscriber eventbus.Subscriber,
	subscriptionStore store.Store,
	goroutinePoolGo GoroutinePoolGoFunc,
) *Server {
	dialTLSConfig := dialCertManager.GetClientTLSConfig()
	listenTLSConfig := listenCertManager.GetServerTLSConfig()
	listenTLSConfig.ClientAuth = tls.NoClientCert

	ln, err := tls.Listen("tcp", config.Addr, &listenTLSConfig)
	if err != nil {
		log.Fatalf("cannot listen and serve: %v", err)
	}

	raConn, err := grpc.Dial(config.ResourceAggregateAddr, grpc.WithTransportCredentials(credentials.NewTLS(&dialTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	raClient := pbRA.NewResourceAggregateClient(raConn)

	rdConn, err := grpc.Dial(config.ResourceDirectoryAddr, grpc.WithTransportCredentials(credentials.NewTLS(&dialTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	rsClient := pbRS.NewResourceShadowClient(rdConn)
	rdClient := pbRD.NewResourceDirectoryClient(rdConn)
	ddClient := pbDD.NewDeviceDirectoryClient(rdConn)

	asConn, err := grpc.Dial(config.AuthServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(&dialTLSConfig)))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	asClient := pbAS.NewAuthorizationServiceClient(asConn)

	if config.DevicesCheckInterval < time.Millisecond*200 {
		log.Fatalf("cannot create server: invalid config.DevicesCheckInterval %v", config.DevicesCheckInterval)
	}

	ctx := context.Background()

	syncPoolHandler := NewGoroutinePoolHandler(goroutinePoolGo, newEventHandler(subscriptionStore, goroutinePoolGo), func(err error) { log.Errorf("%v", err) })
	updateNotificationContainer := raCqrs.NewUpdateNotificationContainer()

	resourceProjection, err := projectionRA.NewProjection(ctx, config.FQDN, resourceEventStore, resourceSubscriber, newResourceCtx(syncPoolHandler, updateNotificationContainer))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	// load subscriptions to projection
	err = subscriptionStore.LoadSubscriptions(ctx, store.SubscriptionQuery{Type: oapiStore.Type_Resource}, newResourceSubscriptionLoader(resourceProjection))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	err = subscriptionStore.LoadSubscriptions(ctx, store.SubscriptionQuery{Type: oapiStore.Type_Device}, newDeviceSubscriptionLoader(resourceProjection))
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	requestHandler := NewRequestHandler(asClient, raClient, rsClient, rdClient, ddClient, resourceProjection, subscriptionStore, updateNotificationContainer, config.TimeoutForRequests)

	devicesSubscription := newDevicesSubscription(requestHandler, goroutinePoolGo)

	server := Server{
		server:              NewHTTP(requestHandler, authInterceptor),
		cfg:                 config,
		handler:             requestHandler,
		ln:                  ln,
		devicesSubscription: devicesSubscription,
	}

	return &server
}

// Serve starts the service's HTTP server and blocks.
func (s *Server) Serve() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.devicesSubscription.Serve(ctx, s.cfg.DevicesCheckInterval)

	return s.server.Serve(s.ln)
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
	return s.server.Shutdown(context.Background())
}
