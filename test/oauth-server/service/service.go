package service

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
)

//Server handle HTTP request
type Service struct {
	server         *http.Server
	requestHandler *RequestHandler
	listener       *listener.Server
}

// New parses configuration and creates new Server with provided store and bus
func New(ctx context.Context, config Config, logger log.Logger) (*Service, error) {
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector, "oauth-server", logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	//tracerProvider := otelClient.GetTracerProvider()

	listener, err := listener.New(config.APIs.HTTP, logger)
	if err != nil {
		otelClient.Close()
		return nil, fmt.Errorf("cannot create http server: %w", err)
	}
	listener.AddCloseFunc(otelClient.Close)

	idTokenPrivateKeyI, err := LoadPrivateKey(config.OAuthSigner.IDTokenKeyFile)
	if err != nil {
		listener.Close()
		return nil, fmt.Errorf("cannot load idTokenKeyFile(%v): %w", config.OAuthSigner.IDTokenKeyFile, err)
	}
	idTokenPrivateKey, ok := idTokenPrivateKeyI.(*rsa.PrivateKey)
	if !ok {
		listener.Close()
		return nil, fmt.Errorf("cannot invalid idTokenKeyFile(%T): %w", idTokenPrivateKey, err)
	}

	accessTokenPrivateKeyI, err := LoadPrivateKey(config.OAuthSigner.AccessTokenKeyFile)
	if err != nil {
		listener.Close()
		return nil, fmt.Errorf("cannot load private accessTokenKeyFile(%v): %w", config.OAuthSigner.AccessTokenKeyFile, err)
	}

	requestHandler, err := NewRequestHandler(ctx, &config, idTokenPrivateKey, accessTokenPrivateKeyI)
	if err != nil {
		listener.Close()
		return nil, fmt.Errorf("cannot create request handler: %w", err)
	}

	server := Service{
		server:         NewHTTP(requestHandler),
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
func (s *Service) Shutdown() error {
	return s.server.Shutdown(context.Background())
}
