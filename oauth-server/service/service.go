package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager"
)

func logError(err error) { log.Error(err) }

//Server handle HTTP request
type Server struct {
	server            *http.Server
	cfg               *Config
	requestHandler    *RequestHandler
	ln                net.Listener
	listenCertManager certManager.CertManager
}

// New parses configuration and creates new Server with provided store and bus
func New(cfg Config) (*Server, error) {
	cfg.SetDefaults()
	log.Info(cfg.String())

	listenCertManager, err := certManager.NewCertManager(cfg.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create listen cert manager: %v", err)
	}
	listenTLSCfg := listenCertManager.GetServerTLSConfig()

	ln, err := tls.Listen("tcp", cfg.Address, listenTLSCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot listen tls and serve: %v", err)
	}

	requestHandler := NewRequestHandler(&cfg)

	server := Server{
		server:            NewHTTP(requestHandler),
		cfg:               &cfg,
		requestHandler:    requestHandler,
		ln:                ln,
		listenCertManager: listenCertManager,
	}

	return &server, nil
}

// Serve starts the service's HTTP server and blocks
func (s *Server) Serve() error {
	return s.server.Serve(s.ln)
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
	if s.listenCertManager != nil {
		s.listenCertManager.Close()
	}
	return s.server.Shutdown(context.Background())
}
