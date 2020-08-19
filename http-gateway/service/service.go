package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"regexp"

	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/kit/log"
	kitNetHttp "github.com/plgd-dev/kit/net/http"
	"github.com/plgd-dev/kit/security/certManager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func logError(err error) { log.Error(err) }

//Server handle HTTP request
type Server struct {
	server            *http.Server
	cfg               *Config
	requestHandler    *RequestHandler
	ln                net.Listener
	listenCertManager certManager.CertManager
	rdConn            *grpc.ClientConn
}

// New parses configuration and creates new Server with provided store and bus
func New(cfg Config) (*Server, error) {
	log.Info(cfg.String())

	listenCertManager, err := certManager.NewCertManager(cfg.Listen)
	if err != nil {
		log.Fatalf("cannot create listen cert manager: %w", err)
	}
	dialCertManager, err := certManager.NewCertManager(cfg.Dial)
	if err != nil {
		log.Fatalf("cannot create dial cert manager: %w", err)
	}
	listenTLSCfg := listenCertManager.GetServerTLSConfig()

	ln, err := tls.Listen("tcp", cfg.Address, listenTLSCfg)
	if err != nil {
		log.Fatalf("cannot listen tls and serve: %w", err)
	}

	rdConn, err := grpc.Dial(
		cfg.ResourceDirectoryAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialCertManager.GetClientTLSConfig())),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	resourceDirectoryClient := pb.NewGrpcGatewayClient(rdConn)
	client, err := client.NewClient("http://localhost", resourceDirectoryClient)
	if err != nil {
		log.Fatalf("cannot initialize new client: %w", err)
	}
	manager, err := NewObservationManager()
	if err != nil {
		log.Fatal("unable to initialize new observation manager %w", err)
	}
	auth := kitNetHttp.NewInterceptor(cfg.JwksURL, dialCertManager.GetClientTLSConfig(), authRules, kitNetHttp.RequestMatcher{
		Method: http.MethodGet,
		URI:    regexp.MustCompile(regexp.QuoteMeta(uri.ClientConfiguration)),
	})
	requestHandler := NewRequestHandler(client, &cfg, manager)

	server := Server{
		server:            NewHTTP(requestHandler, auth),
		cfg:               &cfg,
		requestHandler:    requestHandler,
		ln:                ln,
		listenCertManager: listenCertManager,
		rdConn:            rdConn,
	}

	return &server, nil
}

// Serve starts the service's HTTP server and blocks
func (s *Server) Serve() error {
	return s.server.Serve(s.ln)
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
	observations := s.requestHandler.pop()
	for _, v := range observations {
		for _, s := range v {
			s.OnClose()
		}
	}
	s.rdConn.Close()
	if s.listenCertManager != nil {
		s.listenCertManager.Close()
	}
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
