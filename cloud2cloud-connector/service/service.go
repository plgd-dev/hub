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
	"github.com/plgd-dev/kit/log"
	kitNetHttp "github.com/plgd-dev/kit/net/http"
	"github.com/plgd-dev/kit/security/oauth/manager"
	"google.golang.org/grpc/credentials"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	raService "github.com/plgd-dev/cloud/resource-aggregate/service"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

//Server handle HTTP request
type Server struct {
	server   *http.Server
	cfg      Config
	handler  *RequestHandler
	ln       net.Listener
	cancel   context.CancelFunc
	doneWg   *sync.WaitGroup
	raConn   *grpc.ClientConn
	authConn *grpc.ClientConn
	rdConn   *grpc.ClientConn
}

type DialCertManager = interface {
	GetClientTLSConfig() *tls.Config
}

type ListenCertManager = interface {
	GetServerTLSConfig() *tls.Config
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
	select {
	case <-ctx.Done():
		if ctx.Err() == context.Canceled {
			return false
		}
	}
	return true
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

	ctx, cancel := context.WithCancel(context.Background())
	devicesSubscription := NewDevicesSubscription(ctx, rdClient, raClient, config.ReconnectInterval)
	taskProcessor := NewTaskProcessor(raClient, config.TaskProcessor.MaxParallel, config.TaskProcessor.CacheSize, config.TaskProcessor.Timeout, config.TaskProcessor.Delay)
	subscriptionManager := NewSubscriptionManager(config.EventsURL, asClient, raClient, store, devicesSubscription, config.OAuthCallback, taskProcessor.Trigger, config.ResubscribeInterval)
	requestHandler := NewRequestHandler(config.OAuthCallback, subscriptionManager, asClient, raClient, store, taskProcessor.Trigger)

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
		taskProcessor.Run(ctx, subscriptionManager)
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
		server:   NewHTTP(requestHandler, auth),
		cfg:      config,
		handler:  requestHandler,
		ln:       ln,
		cancel:   cancel,
		doneWg:   &wg,
		raConn:   raConn,
		rdConn:   rdConn,
		authConn: authConn,
	}

	return &server
}

// Serve starts the service's HTTP server and blocks.
func (s *Server) Serve() error {
	defer func() {
		s.raConn.Close()
		s.rdConn.Close()
		s.authConn.Close()
	}()
	return s.server.Serve(s.ln)
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
	s.cancel()
	s.doneWg.Wait()
	return s.server.Shutdown(context.Background())
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
