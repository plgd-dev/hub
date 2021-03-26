package service

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/plgd-dev/cloud/authorization/persistence/mongodb"
	"github.com/plgd-dev/cloud/authorization/uri"

	"github.com/patrickmn/go-cache"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"golang.org/x/sync/errgroup"

	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/authorization/persistence"
	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/pkg/net/listener"
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
	ownerClaim     string
}

// Server is an HTTP server for the Service.
type Server struct {
	service    *Service
	grpcServer *server.Server
	httpServer *fasthttp.Server
	cfg        Config
	listener   net.Listener
}

func NewService(deviceProvider, sdkProvider Provider, persistence Persistence, ownerClaim string) *Service {
	return &Service{
		deviceProvider: deviceProvider,
		sdkProvider:    sdkProvider,
		persistence:    persistence,
		csrfTokens:     cache.New(5*time.Minute, 10*time.Minute),
		ownerClaim:     ownerClaim,
	}
}

func NewServer(ctx context.Context, cfg Config, logger *zap.Logger, deviceProvider Provider, sdkProvider Provider) (*Server, error) {
	grpcServer, err := server.New(cfg.Service.GRPC, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create grpc listener: %w", err)
	}

	httpListener, err := listener.New(cfg.Service.HTTP, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create http listener: %w", err)
	}

	persistence, err := mongodb.NewStore(ctx, cfg.Databases.MongoDB, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create connector to mongo: %w", err)
	}
	grpcServer.AddCloseFunc(func() { persistence.Close(ctx) })

	service := NewService(deviceProvider, sdkProvider, persistence, cfg.Clients.Device.OwnerClaim)

	pb.RegisterAuthorizationServiceServer(grpcServer.Server, service)

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

	return &Server{service: service, grpcServer: grpcServer, httpServer: httpServer, cfg: cfg, listener: httpListener}, nil
}

// New creates the service's HTTP server.
func New(ctx context.Context, cfg Config, logger *zap.Logger) (*Server, error) {
	deviceProvider, err := provider.New(cfg.Clients.Device, logger, "query", "offline", "code")
	if err != nil {
		return nil, fmt.Errorf("cannot create device provider: %w", err)
	}
	sdkProvider, err := provider.New(provider.Config{
		Provider: "generic",
		Config:   cfg.Clients.SDK.Config,
		HTTP:     cfg.Clients.SDK.HTTP,
	}, logger, "form_post", "online", "token")
	if err != nil {
		return nil, fmt.Errorf("cannot create sdk provider: %w", err)
	}

	s, err := NewServer(ctx, cfg, logger, deviceProvider, sdkProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot create server: %w", err)
	}
	s.grpcServer.AddCloseFunc(deviceProvider.Close)
	s.grpcServer.AddCloseFunc(sdkProvider.Close)
	return s, nil
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
	s.grpcServer.Close()
	s.httpServer.Shutdown()
}
