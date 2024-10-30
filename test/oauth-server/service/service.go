package service

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
)

const serviceName = "mock-oauth-server"

// Server handle HTTP request
type Service struct {
	server         *http.Server
	requestHandler *RequestHandler
	listener       *listener.Server
}

// New parses configuration and creates new Server with provided store and bus
func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector.Config, serviceName, fileWatcher, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	otelClient.AddCloseFunc(cancel)
	tracerProvider := otelClient.GetTracerProvider()

	listener, err := listener.New(config.APIs.HTTP.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		otelClient.Close()
		return nil, fmt.Errorf("cannot create http server: %w", err)
	}
	listener.AddCloseFunc(otelClient.Close)
	closeListener := func() {
		if errC := listener.Close(); errC != nil {
			logger.Errorf("cannot close listener: %w", errC)
		}
	}

	idTokenPrivateKeyI, err := LoadPrivateKey(config.OAuthSigner.IDTokenKeyFile)
	if err != nil {
		closeListener()
		return nil, fmt.Errorf("cannot load idTokenKeyFile(%v): %w", config.OAuthSigner.IDTokenKeyFile, err)
	}
	idTokenPrivateKey, ok := idTokenPrivateKeyI.(*rsa.PrivateKey)
	if !ok {
		closeListener()
		return nil, fmt.Errorf("cannot invalid idTokenKeyFile(%T): %w", idTokenPrivateKey, err)
	}

	accessTokenPrivateKeyI, err := LoadPrivateKey(config.OAuthSigner.AccessTokenKeyFile)
	if err != nil {
		closeListener()
		return nil, fmt.Errorf("cannot load private accessTokenKeyFile(%v): %w", config.OAuthSigner.AccessTokenKeyFile, err)
	}

	requestHandler, err := NewRequestHandler(ctx, &config, idTokenPrivateKey, accessTokenPrivateKeyI)
	if err != nil {
		closeListener()
		return nil, fmt.Errorf("cannot create request handler: %w", err)
	}

	httpServer := http.Server{
		Handler:           kitNetHttp.OpenTelemetryNewHandler(NewHTTP(requestHandler, logger), serviceName, tracerProvider),
		ReadTimeout:       config.APIs.HTTP.Server.ReadTimeout,
		ReadHeaderTimeout: config.APIs.HTTP.Server.ReadHeaderTimeout,
		WriteTimeout:      config.APIs.HTTP.Server.WriteTimeout,
		IdleTimeout:       config.APIs.HTTP.Server.IdleTimeout,
	}

	server := Service{
		server:         &httpServer,
		requestHandler: requestHandler,
		listener:       listener,
	}

	return &server, nil
}

// Serve starts the service's HTTP server and blocks
func (s *Service) Serve() error {
	return s.server.Serve(s.listener)
}

// Shutdown ends serving
func (s *Service) Close() error {
	return s.server.Shutdown(context.Background())
}
