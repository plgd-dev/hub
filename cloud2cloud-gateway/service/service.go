package service

import (
	"context"
	"crypto/tls"
	"go.uber.org/zap"
	"net"
	"net/http"
	"regexp"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	storeMongodb "github.com/plgd-dev/cloud/cloud2cloud-gateway/store/mongodb"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	kitNetHttp "github.com/plgd-dev/cloud/pkg/net/http"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	raClient "github.com/plgd-dev/cloud/resource-aggregate/client"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
)

//Server handle HTTP request
type Server struct {
	server  *http.Server
	cfg     Config
	handler *RequestHandler
	ln      net.Listener
	rdConn  *grpc.ClientConn

	httpCertManager *server.CertManager
	dbCertManager *client.CertManager
	oauthCertManager *client.CertManager
	rdCertManager *client.CertManager
	raCertManager *client.CertManager

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
func New(config Config, logger *zap.Logger) *Server {
	dbCertManager, err := client.New(config.Database.MongoDB.TLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create db dial cert manager %v", err)
		return nil
	}
	subscriptionStore, err := storeMongodb.NewStore(context.Background(), config.Database.MongoDB, storeMongodb.WithTLS(dbCertManager.GetTLSConfig()))
	if err != nil {
		log.Fatalf("cannot create mongodb substore %w", err)
		return nil
	}

	httpCertManager, err := server.New(config.Service.Http.TLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create http listen cert manager %v", err)
		return nil
	}
	ln, err := tls.Listen("tcp", config.Service.Http.Addr, httpCertManager.GetTLSConfig())
	if err != nil {
		log.Fatalf("cannot listen and serve: %v", err)
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

	authInterceptor := kitNetHttp.NewInterceptor(config.Clients.OAuthProvider.JwksURL, oauthTLSConfig, authRules)
	oauthMgr, err := manager.NewManagerFromConfiguration(config.Clients.OAuthProvider.OAuth, oauthTLSConfig)
	if err != nil {
		log.Fatalf("cannot create oauth manager: %v", err)
	}

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

	pool, err := ants.NewPool(config.Service.Capabilities.GoRoutinePoolSize)
	if err != nil {
		log.Fatalf("cannot create goroutine pool: %v", err)
	}

	raCertManager, err := client.New(config.Clients.ResourceAggregate.TLSConfig, logger)
	if err != nil {
		log.Fatalf("cannot create resource-directory dial cert manager %v", err)
	}
	resourceSubscriber, err := nats.NewSubscriber(config.Clients.Nats, pool.Submit, func(err error) {
		log.Errorf("error occurs during receiving event: %v", err) }, nats.WithTLS(raCertManager.GetTLSConfig()))
	if err != nil {
		log.Fatalf("cannot create eventbus subscriber: %v", err)
	}
	raConn, err := grpc.Dial(
		config.Clients.ResourceAggregate.Addr,
		grpc.WithTransportCredentials(credentials.NewTLS(raCertManager.GetTLSConfig())),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		log.Fatalf("cannot connect to resource aggregate: %v", err)
	}
	raClient := raClient.New(raConn, resourceSubscriber)

	emitEvent := createEmitEventFunc(raCertManager.GetTLSConfig(), config.Service.Capabilities.EmitEventTimeout)
	ctx, cancel := context.WithCancel(context.Background())
	subMgr := NewSubscriptionManager(ctx, subscriptionStore, rdClient, config.Service.Capabilities.ReconnectInterval, emitEvent)
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

	requestHandler := NewRequestHandler(rdClient, raClient, subMgr, emitEvent, config.Clients.OAuthProvider.OwnerClaim)

	server := Server{
		server:  NewHTTP(requestHandler, authInterceptor),
		cfg:     config,
		handler: requestHandler,
		ln:      ln,
		rdConn:  rdConn,
		httpCertManager: httpCertManager,
		dbCertManager: dbCertManager,
		oauthCertManager: oauthCertManager,
		rdCertManager: rdCertManager,
		raCertManager: raCertManager,
		cancel:  cancel,
		doneWg:  &wg,
	}

	return &server
}

// Serve starts the service's HTTP server and blocks.
func (s *Server) Serve() error {
	defer func() {
		s.doneWg.Wait()

		s.httpCertManager.Close()
		s.dbCertManager.Close()
		if s.oauthCertManager != nil { s.oauthCertManager.Close() }
		s.rdCertManager.Close()
		s.raCertManager.Close()

		s.rdConn.Close()
	}()
	return s.server.Serve(s.ln)
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
	s.cancel()
	return s.server.Shutdown(context.Background())
}

// https://openconnectivity.org/draftspecs/Gaborone/OCF_Cloud_API_for_Cloud_Services.pdf
var authRules = map[string][]kitNetHttp.AuthArgs{
	http.MethodGet: {
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]*\?` + ContentQuery + `=` + ContentQueryBaseValue + `$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]*\?` + ContentQuery + `=` + ContentQueryAllValue + `$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
				regexp.MustCompile(`r:resources:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]*\?` + ContentQuery + `=` + ContentQueryBaseValue + `$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]*\?` + ContentQuery + `=` + ContentQueryAllValue + `$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
				regexp.MustCompile(`r:resources:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]+.*[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:resources:.*`),
			},
		},
	},
	http.MethodPost: {
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+subscriptions[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
				regexp.MustCompile(`w:subscriptions:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]+subscriptions[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
				regexp.MustCompile(`w:subscriptions:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]+.*[\/]+subscriptions[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:resources:.*`),
				regexp.MustCompile(`w:subscriptions:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]+.*[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`w:resources:.*`),
			},
		},
	},
	http.MethodDelete: {
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+subscriptions[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
				regexp.MustCompile(`w:subscriptions:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]+subscriptions[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:deviceinformation:.*`),
				regexp.MustCompile(`w:subscriptions:.*`),
			},
		},
		{
			URI: regexp.MustCompile(`[\/]+api[\/]+v1[\/]+devices[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]+.*[\/]+subscriptions[\/]+[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}[\/]*$`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`r:resources:.*`),
				regexp.MustCompile(`w:subscriptions:.*`),
			},
		},
	},
}
