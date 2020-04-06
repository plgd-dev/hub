package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/go-ocf/kit/security/certManager"

	"github.com/go-ocf/ocf-cloud/authorization/uri"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/patrickmn/go-cache"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"golang.org/x/sync/errgroup"

	"github.com/go-ocf/ocf-cloud/authorization/pb"
	"github.com/go-ocf/ocf-cloud/authorization/persistence"
	"github.com/go-ocf/ocf-cloud/authorization/provider"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
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
	serverCertManager certManager.CertManager
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
func New(cfg Config, persistence Persistence, deviceProvider, sdkProvider provider.Provider) (*Server, error) {
	serverCertManager, err := certManager.NewCertManager(cfg.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create server cert manager %v", err)
	}
	httpServerTLSConfig := serverCertManager.GetServerTLSConfig()
	httpServerTLSConfig.ClientAuth = tls.NoClientCert
	listener, err := tls.Listen("tcp", cfg.HTTPAddr, &httpServerTLSConfig)
	if err != nil {
		return nil, fmt.Errorf("listening failed: %v", err)
	}

	service := newService(deviceProvider, sdkProvider, persistence)
	serverTLSConfig := serverCertManager.GetServerTLSConfig()
	server, err := kitNetGrpc.NewServer(cfg.Addr, grpc.Creds(credentials.NewTLS(&serverTLSConfig)))
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
		Handler: httpRouter.Handler,
		IdleTimeout: time.Second,
	}

	return &Server{service: service, grpcServer: server, httpServer: httpServer, cfg: cfg, serverCertManager: serverCertManager, listener: listener}, nil
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
	s.serverCertManager.Close()
}
