package service

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sync"

	"google.golang.org/grpc"

	connectorStore "github.com/plgd-dev/cloud/cloud2cloud-connector/store"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/uri"
	kitNetHttp "github.com/plgd-dev/cloud/pkg/net/http"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"github.com/plgd-dev/kit/log"
	"google.golang.org/grpc/credentials"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	raService "github.com/plgd-dev/cloud/resource-aggregate/service"
)

//Server handle HTTP request
type Server struct {
	server            *http.Server
	cfg               Config
	handler           *RequestHandler
	ln                net.Listener
	cancel            context.CancelFunc
	doneWg            *sync.WaitGroup
	raConn            *grpc.ClientConn
	authConn          *grpc.ClientConn
	rdConn            *grpc.ClientConn
	dialCertManager   DialCertManager
	listenCertManager ListenCertManager
	db                connectorStore.Store
	sub               *subscriber.Subscriber
}

type DialCertManager = interface {
	GetClientTLSConfig() *tls.Config
	Close()
}

type ListenCertManager = interface {
	GetServerTLSConfig() *tls.Config
	Close()
}

func runDevicePulling(ctx context.Context,
	config Config,
	s *Store,
	asClient pbAS.AuthorizationServiceClient,
	raClient raService.ResourceAggregateClient,
	devicesSubscription *DevicesSubscription,
	subscriptionManager *SubscriptionManager,
	triggerTask func(Task),
) bool {
	ctx, cancel := context.WithTimeout(ctx, config.PullDevicesInterval)
	defer cancel()
	err := pullDevices(ctx, s, asClient, raClient, devicesSubscription, subscriptionManager, config.OAuthCallback, triggerTask)
	if err != nil {
		log.Errorf("cannot pull devices: %v", err)
	}
	<-ctx.Done()
	return ctx.Err() != context.Canceled
}

//New create new Server with provided store and bus
func New(config Config, dialCertManager DialCertManager, listenCertManager ListenCertManager, db connectorStore.Store) *Server {
	dialTLSConfig := dialCertManager.GetClientTLSConfig()
	var ln net.Listener
	var err error
	if listenCertManager != nil {
		listenTLSConfig := listenCertManager.GetServerTLSConfig()
		ln, err = tls.Listen("tcp", config.Addr, listenTLSConfig)
		if err != nil {
			log.Fatalf("cannot listen and serve: %v", err)
		}
	} else {
		ln, err = net.Listen("tcp", config.Addr)
		if err != nil {
			log.Fatalf("cannot listen and serve: %v", err)
		}
	}

	oauthMgr, err := manager.NewManagerFromConfiguration(config.OAuth, dialTLSConfig)
	if err != nil {
		log.Fatalf("cannot create oauth manager: %v", err)
	}
	defer func() {
		oauthMgr.Close()
	}()

	raConn, err := grpc.Dial(config.ResourceAggregateAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	raClient := raService.NewResourceAggregateClient(raConn)

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
	store, err := NewStore(context.Background(), db)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	logger, err := log.NewLogger(log.Config{})
	if err != nil {
		log.Fatalf("cannot create logger: %v", err)
	}

	sub, err := subscriber.New(config.Nats, logger.Sugar(), subscriber.WithUnmarshaler(utils.Unmarshal))
	if err != nil {
		log.Fatalf("cannot create subscriber: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	devicesSubscription := NewDevicesSubscription(ctx, rdClient, raClient, sub, config.ReconnectInterval)
	taskProcessor := NewTaskProcessor(raClient, config.TaskProcessor.MaxParallel, config.TaskProcessor.CacheSize, config.TaskProcessor.Timeout, config.TaskProcessor.Delay)
	subscriptionManager := NewSubscriptionManager(config.EventsURL, asClient, raClient, store, devicesSubscription, config.OAuthCallback, taskProcessor.Trigger, config.ResubscribeInterval)
	requestHandler := NewRequestHandler(config.OAuthCallback, subscriptionManager, asClient, raClient, store, taskProcessor.Trigger, config.OwnerClaim)

	var wg sync.WaitGroup
	if !config.PullDevicesDisabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for runDevicePulling(ctx, config, store, asClient, raClient, devicesSubscription, subscriptionManager, taskProcessor.Trigger) {
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := taskProcessor.Run(ctx, subscriptionManager); err != nil {
			if !kitNetGrpc.IsContextCanceled(err) {
				log.Errorf("failed to process subscriptionManager tasks: %w", err)
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		subscriptionManager.Run(ctx)
	}()

	oauthURL, _ := url.Parse(config.OAuthCallback)
	auth := kitNetHttp.NewInterceptor(config.JwksURL, dialCertManager.GetClientTLSConfig(), authRules, kitNetHttp.RequestMatcher{
		Method: http.MethodGet,
		URI:    regexp.MustCompile(regexp.QuoteMeta(oauthURL.Path)),
	}, kitNetHttp.RequestMatcher{
		Method: http.MethodPost,
		URI:    regexp.MustCompile(regexp.QuoteMeta(oauthURL.Path)),
	}, kitNetHttp.RequestMatcher{
		Method: http.MethodPost,
		URI:    regexp.MustCompile(regexp.QuoteMeta(uri.Events)),
	},
	)
	server := Server{
		server:            NewHTTP(requestHandler, auth),
		cfg:               config,
		handler:           requestHandler,
		ln:                ln,
		cancel:            cancel,
		doneWg:            &wg,
		raConn:            raConn,
		rdConn:            rdConn,
		authConn:          authConn,
		dialCertManager:   dialCertManager,
		listenCertManager: listenCertManager,
		db:                db,
		sub:               sub,
	}

	return &server
}

// Serve starts the service's HTTP server and blocks.
func (s *Server) Serve() error {
	defer func() {
		s.sub.Close()
		if err := s.raConn.Close(); err != nil {
			log.Errorf("failed to close ResourceAggregate connection: %v", err)
		}
		if err := s.rdConn.Close(); err != nil {
			log.Errorf("failed to close ResourceDirectory connection: %v", err)
		}
		if err := s.authConn.Close(); err != nil {
			log.Errorf("failed to close Grpc connection: %v", err)
		}
		s.dialCertManager.Close()
		if s.listenCertManager != nil {
			s.listenCertManager.Close()
		}
		if err := s.db.Close(context.Background()); err != nil {
			log.Errorf("failed to close db connector: %v", err)
		}
	}()
	return s.server.Serve(s.ln)
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
	s.cancel()
	s.doneWg.Wait()
	return s.server.Close()
}

var authRules = map[string][]kitNetHttp.AuthArgs{
	http.MethodGet: {
		{
			URI: regexp.MustCompile(regexp.QuoteMeta(uri.API) + `\/.*`),
		},
	},
	http.MethodPost: {
		{
			URI: regexp.MustCompile(regexp.QuoteMeta(uri.API) + `\/.*`),
		},
	},
	http.MethodDelete: {
		{
			URI: regexp.MustCompile(regexp.QuoteMeta(uri.API) + `\/.*`),
		},
	},
	http.MethodPut: {
		{
			URI: regexp.MustCompile(regexp.QuoteMeta(uri.API) + `\/.*`),
		},
	},
}
