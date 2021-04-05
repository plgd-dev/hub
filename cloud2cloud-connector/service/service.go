package service

import (
	"context"
	"crypto/tls"
	"go.uber.org/zap"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	storeMongodb "github.com/plgd-dev/cloud/cloud2cloud-connector/store/mongodb"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/uri"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	kitNetHttp "github.com/plgd-dev/cloud/pkg/net/http"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	raService "github.com/plgd-dev/cloud/resource-aggregate/service"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
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

	httpCertManager *server.CertManager
	dbCertManager   *client.CertManager
	asCertManager   *client.CertManager
	raCertManager   *client.CertManager
	rdCertManager   *client.CertManager
	oauthCertManager *client.CertManager
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
	ctx, cancel := context.WithTimeout(ctx, config.Service.Capabilities.PullDevicesInterval)
	defer cancel()
	err := pullDevices(ctx, s, asClient, raClient, devicesSubscription, subscriptionManager, config.Service.Http.OAuthCallback, triggerTask)
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
func New(config Config, logger *zap.Logger) *Server {
	dbCertManager, err := client.New(config.Database.MongoDB.TLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create db dial cert manager %v", err)
	}
	db, err := storeMongodb.NewStore(context.Background(), config.Database.MongoDB, storeMongodb.WithTLS(dbCertManager.GetTLSConfig()))
	if err != nil {
		log.Fatalf("cannot create mongodb store %v", err)
		//return nil
	}

	var ln net.Listener
	httpCertManager, err := server.New(config.Service.Http.TLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create http listen cert manager %v", err)
		ln, err = net.Listen("tcp", config.Service.Http.Addr)
		if err != nil {
			log.Fatalf("cannot listen and serve: %v", err)
		}
	} else {
		ln, err = tls.Listen("tcp", config.Service.Http.Addr, httpCertManager.GetTLSConfig())
		if err != nil {
			log.Fatalf("cannot listen and serve with tls: %v", err)
		}
	}

	var oauthCertManager *client.CertManager = nil
	var oauthTLSConfig *tls.Config = nil
	err = config.Clients.OAuthProvider.TLSConfig.Validate()
	if err != nil {
		log.Errorf("failed to validate client tls config: %v", err)
	} else {
		oauthCertManager, err := client.New(config.Clients.OAuthProvider.TLSConfig, logger)
		if err != nil {
			log.Errorf("cannot create oauth client cert manager %v", err)
		} else {
			oauthTLSConfig = oauthCertManager.GetTLSConfig()
		}
	}

	oauthMgr, err := manager.NewManagerFromConfiguration(config.Clients.OAuthProvider.OAuth, oauthTLSConfig)
	if err != nil {
		log.Fatalf("cannot create oauth manager: %v", err)
	}

	raCertManager, err := client.New(config.Clients.ResourceAggregate.TLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create resource-aggregate dial cert manager %v", err)
	}
	raConn, err := grpc.Dial(config.Clients.ResourceAggregate.Addr,
		grpc.WithTransportCredentials(credentials.NewTLS(raCertManager.GetTLSConfig())),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	raClient := raService.NewResourceAggregateClient(raConn)

	asCertManager, err := client.New(config.Clients.Authorization.TLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create authorization dial cert manager %v", err)
	}
	authConn, err := grpc.Dial(config.Clients.Authorization.Addr,
		grpc.WithTransportCredentials(credentials.NewTLS(asCertManager.GetTLSConfig())),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	asClient := pbAS.NewAuthorizationServiceClient(authConn)

	rdCertManager, err := client.New(config.Clients.ResourceDirectory.TLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create resource-directory dial cert manager %v", err)
	}
	rdConn, err := grpc.Dial(config.Clients.ResourceDirectory.Addr,
		grpc.WithTransportCredentials(credentials.NewTLS(rdCertManager.GetTLSConfig())),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	rdClient := pbGRPC.NewGrpcGatewayClient(rdConn)

	_, err = url.Parse(config.Service.Http.OAuthCallback)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}
	store, err := NewStore(context.Background(), db)
	if err != nil {
		log.Fatalf("cannot create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	devicesSubscription := NewDevicesSubscription(ctx, rdClient, raClient, config.Service.Capabilities.ReconnectInterval)
	taskProcessor := NewTaskProcessor(raClient, config.Service.TaskProcessor.MaxParallel, config.Service.TaskProcessor.CacheSize, config.Service.TaskProcessor.Timeout, config.Service.TaskProcessor.Delay)
	subscriptionManager := NewSubscriptionManager(config.Service.Http.EventsURL, asClient, raClient, store, devicesSubscription, config.Service.Http.OAuthCallback, taskProcessor.Trigger, config.Service.Capabilities.ResubscribeInterval)
	requestHandler := NewRequestHandler(config.Service.Http.OAuthCallback, subscriptionManager, asClient, raClient, store, taskProcessor.Trigger, config.Clients.OAuthProvider.OwnerClaim)

	var wg sync.WaitGroup
	if !config.Service.Capabilities.PullDevicesDisabled {
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

	oauthURL, _ := url.Parse(config.Service.Http.OAuthCallback)
	auth := kitNetHttp.NewInterceptor(config.Clients.OAuthProvider.JwksURL, oauthTLSConfig, authRules, kitNetHttp.RequestMatcher{
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

		httpCertManager: httpCertManager,
		dbCertManager: dbCertManager,
		asCertManager: asCertManager,
		raCertManager: raCertManager,
		rdCertManager: rdCertManager,
		oauthCertManager: oauthCertManager,
	}

	return &server
}

// Serve starts the service's HTTP server and blocks.
func (s *Server) Serve() error {
	defer func() {
		s.raConn.Close()
		s.rdConn.Close()
		s.authConn.Close()

		s.httpCertManager.Close()
		s.dbCertManager.Close()
		s.asCertManager.Close()
		s.raCertManager.Close()
		s.rdCertManager.Close()
		if s.oauthCertManager != nil { s.oauthCertManager.Close() }
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
