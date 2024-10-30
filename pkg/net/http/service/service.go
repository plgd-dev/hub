package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpJwt "github.com/plgd-dev/hub/v2/pkg/net/http/jwt"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
)

// Service handle HTTP request
type Service struct {
	server   *http.Server
	config   *Config
	listener *listener.Server
	router   *mux.Router
	serving  atomic.Bool
}

// New parses configuration and creates new http service
func New(config Config) (*Service, error) {
	listener, err := listener.New(config.HTTPConnection, config.FileWatcher, config.Logger, config.TraceProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot create grpc server: %w", err)
	}

	router := mux.NewRouter()
	auth := pkgHttpJwt.NewInterceptorWithValidator(config.Validator, config.AuthRules, config.WhiteEndpointList...)
	r0 := serverMux.NewRouter(config.QueryCaseInsensitive, auth, pkgHttp.WithLogger(config.Logger))
	r0.PathPrefix("/").Handler(router)

	httpServer := http.Server{
		Handler:           pkgHttp.OpenTelemetryNewHandler(r0, config.ServiceName, config.TraceProvider),
		ReadTimeout:       config.HTTPServer.ReadTimeout,
		ReadHeaderTimeout: config.HTTPServer.ReadHeaderTimeout,
		WriteTimeout:      config.HTTPServer.WriteTimeout,
		IdleTimeout:       config.HTTPServer.IdleTimeout,
	}

	server := Service{
		server:   &httpServer,
		config:   &config,
		listener: listener,
		router:   router,
	}

	return &server, nil
}

// Serve starts the service's HTTP server and blocks
func (s *Service) Serve() error {
	if !s.serving.CompareAndSwap(false, true) {
		return errors.New("service is already serving")
	}
	err := s.server.Serve(s.listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Close ends serving
func (s *Service) Close() error {
	if !s.serving.Load() {
		return s.listener.Close()
	}
	return s.server.Shutdown(context.Background())
}

func (s *Service) GetRouter() *mux.Router {
	return s.router
}

func (s *Service) AddCloseFunc(f func()) {
	s.listener.AddCloseFunc(f)
}
