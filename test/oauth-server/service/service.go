package service

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"

	"github.com/plgd-dev/hub/pkg/log"

	"github.com/plgd-dev/hub/pkg/net/listener"
)

//Server handle HTTP request
type Service struct {
	server         *http.Server
	requestHandler *RequestHandler
	listener       *listener.Server
}

// New parses configuration and creates new Server with provided store and bus
func New(ctx context.Context, config Config, logger log.Logger) (*Service, error) {
	listener, err := listener.New(config.APIs.HTTP, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create http server: %w", err)
	}

	idTokenPrivateKeyI, err := LoadPrivateKey(config.OAuthSigner.IDTokenKeyFile)
	if err != nil {
		return nil, fmt.Errorf("cannot load idTokenKeyFile(%v): %v", config.OAuthSigner.IDTokenKeyFile, err)
	}
	idTokenPrivateKey, ok := idTokenPrivateKeyI.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("cannot invalid idTokenKeyFile(%T): %v", idTokenPrivateKey, err)
	}

	accessTokenPrivateKeyI, err := LoadPrivateKey(config.OAuthSigner.AccessTokenKeyFile)
	if err != nil {
		return nil, fmt.Errorf("cannot load private accessTokenKeyFile(%v): %v", config.OAuthSigner.AccessTokenKeyFile, err)
	}

	requestHandler, err := NewRequestHandler(&config, idTokenPrivateKey, accessTokenPrivateKeyI)
	if err != nil {
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
