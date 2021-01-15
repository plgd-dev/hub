package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/plgd-dev/cloud/authorization/persistence/mongodb"
	"github.com/plgd-dev/kit/log"
	"net"
	"time"

	"github.com/plgd-dev/kit/security/certManager"

	"github.com/plgd-dev/cloud/authorization/uri"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/patrickmn/go-cache"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"golang.org/x/sync/errgroup"

	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/authorization/persistence"
	"github.com/plgd-dev/cloud/authorization/provider"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

// Provider defines interface for authentification against auth service
type Provider = interface {
	Exchange(ctx context.Context, authorizationProvider, authorizationCode string) (*provider.Token, error)
	Refresh(ctx context.Context, refreshToken string) (*provider.Token, error)
	AuthCodeURL(csrfToken string) string
}

// Provider defines interface for authentification against auth service
type Persistence = interface {
	NewTransaction(ctx context.Context) persistence.PersistenceTx
	Clear(ctx context.Context) error
	Close(ctx context.Context) error
}

// Service holds dependencies of the authorization Service.
type Service struct {
	pb.UnimplementedAuthorizationServiceServer
	deviceProvider Provider
	sdkProvider    Provider
	persistence    Persistence
	csrfTokens     *cache.Cache
}

// Server is an HTTP server for the Service.
type Server struct {
	service           *Service
	grpcServer        *kitNetGrpc.Server
	httpServer        *fasthttp.Server
	cfg               Config
	grpcCertManager   certManager.ServerCertManager
	httpCertManager   certManager.ServerCertManager
	mongoCertManager  certManager.ClientCertManager
	listener          net.Listener
}

func newService(deviceProvider, sdkProvider Provider, persistence Persistence) *Service {
	return &Service{
		deviceProvider: deviceProvider,
		sdkProvider:    sdkProvider,
		persistence:    persistence,
		csrfTokens:     cache.New(5*time.Minute, 10*time.Minute),
	}
}

// New creates the service's HTTP server.
func New(cfg Config) (*Server, error) {

	mongoCertManager, err := certManager.NewClientCertManager(cfg.Database.MongoDB.TLSConfig)
	if err != nil {
		log.Fatalf("cannot parse config: %v", err)
	}

	mongoTlsConfig := mongoCertManager.GetClientTLSConfig()
	persistence, err := mongodb.NewStore(context.Background(), cfg.Database.MongoDB, mongodb.WithTLS(mongoTlsConfig))
	if err != nil {
		log.Fatalf("cannot parse config: %v", err)
	}
	if cfg.Clients.Device.OAuth2.AccessType == "" {
		cfg.Clients.Device.OAuth2.AccessType = "offline"
	}
	if cfg.Clients.Device.OAuth2.ResponseType == "" {
		cfg.Clients.Device.OAuth2.ResponseType = "code"
	}
	if cfg.Clients.Device.OAuth2.ResponseMode == "" {
		cfg.Clients.Device.OAuth2.ResponseMode = "query"
	}
	if cfg.Clients.SDK.OAuth.AccessType == "" {
		cfg.Clients.SDK.OAuth.AccessType = "online"
	}
	if cfg.Clients.SDK.OAuth.ResponseType == "" {
		cfg.Clients.SDK.OAuth.ResponseType = "token"
	}
	if cfg.Clients.SDK.OAuth.ResponseMode == "" {
		cfg.Clients.SDK.OAuth.ResponseMode = "query"
	}
	deviceProvider := provider.New(cfg.Clients.Device)
	sdkProvider := provider.New(provider.Config{
		Provider: "generic",
		OAuth2:   cfg.Clients.SDK.OAuth,
	})
	service := newService(deviceProvider, sdkProvider, persistence)

	httpCertManager, err := certManager.NewServerCertManager(cfg.Service.HttpServer.HttpTLSConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot create http server cert manager %w", err)
	}
	httpServerTLSConfig := httpCertManager.GetServerTLSConfig()
	//httpServerTLSConfig.ClientAuth = tls.NoClientCert // TODO : no need to use due to tlsConfig which set by verifyClientCertificate of ServerConfig
	listener, err := tls.Listen("tcp", cfg.Service.HttpServer.HttpAddr, httpServerTLSConfig)
	if err != nil {
		return nil, fmt.Errorf("listening failed: %w", err)
	}
	grpcCertManager, err := certManager.NewServerCertManager(cfg.Service.GrpcServer.GrpcTLSConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot create grpc server cert manager %w", err)
	}
	grpcServerTLSConfig := grpcCertManager.GetServerTLSConfig()
	server, err := kitNetGrpc.NewServer(cfg.Service.GrpcServer.GrpcAddr, grpc.Creds(credentials.NewTLS(grpcServerTLSConfig)))
	if err != nil {
		return nil, err
	}

	pb.RegisterAuthorizationServiceServer(server.Server, service)

	httpRouter := fasthttprouter.New()
	httpRouter.GET(uri.AuthorizationCode, service.HandleAuthorizationCode)
	httpRouter.POST(uri.AuthorizationCode, service.HandleAuthorizationCode)
	httpRouter.GET(uri.AccessToken, service.HandleAccessToken)
	httpRouter.POST(uri.AccessToken, service.HandleAccessToken)
	httpRouter.GET(uri.OAuthCallback, service.HandleOAuthCallback)
	httpRouter.POST(uri.OAuthCallback, service.HandleOAuthCallback)
	httpRouter.GET(uri.Healthcheck, service.Healthcheck)
	httpRouter.GET(uri.JWKs, service.HandleJWKs)
	httpServer := &fasthttp.Server{
		Handler:     httpRouter.Handler,
		IdleTimeout: time.Second,
	}

	return &Server{service: service, grpcServer: server, httpServer: httpServer, cfg: cfg,
		grpcCertManager: grpcCertManager, httpCertManager: httpCertManager, mongoCertManager: mongoCertManager, listener: listener}, nil
}

// Serve starts the service's GRPC and HTTP server and blocks.
func (s *Server) Serve() error {
	g := new(errgroup.Group)
	g.Go(func() error { return s.grpcServer.Serve() })
	g.Go(func() error { return s.httpServer.Serve(s.listener) })

	g.Wait()
	return nil
}

// Shutdown ends serving
func (s *Server) Shutdown() {
	s.grpcServer.Stop()
	s.httpServer.Shutdown()
	s.listener.Close()
	s.grpcCertManager.Close()
	s.httpCertManager.Close()
	s.mongoCertManager.Close()
}
