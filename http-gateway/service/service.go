package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"regexp"

	pbCA "github.com/plgd-dev/cloud/certificate-authority/pb"
	GrpcGWClient "github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/kit/log"
	kitNetHttp "github.com/plgd-dev/kit/net/http"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
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
	rdConn            *grpc.ClientConn
	caConn            *grpc.ClientConn

	httpCertManager   *server.CertManager
	rdCertManager     *client.CertManager
	caCertManager     *client.CertManager
	oauthCertManager  *client.CertManager
}

// New parses configuration and creates new Server with provided store and bus
func New(config Config) (*Server, error) {
	logger, err := log.NewLogger(config.Log)
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}

	log.Set(logger)
	log.Info(config.String())

	config = config.checkForDefaults()
	httpCertManager, err := server.New(config.Service.HttpConfig.HttpTLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create http server cert manager %w", err)
	}
	ln, err := tls.Listen("tcp", config.Service.HttpConfig.HttpAddr, httpCertManager.GetTLSConfig())
	if err != nil {
		log.Fatalf("cannot listen tls and serve: %v", err)
	}

	rdCertManager, err := client.New(config.Clients.RDConfig.ResourceDirectoryTLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create rd client cert manager %w", err)
	}
	rdConn, err := grpc.Dial(
		config.Clients.RDConfig.ResourceDirectoryAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(rdCertManager.GetTLSConfig())),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	//TODO : need to check below logics
	resourceDirectoryClient := pb.NewGrpcGatewayClient(rdConn)
	grpcClient, err := GrpcGWClient.NewClient("http://localhost", resourceDirectoryClient)
	if err != nil {
		log.Fatalf("cannot initialize new client: %v", err)
	}

	caCertManager, err := client.New(config.Clients.CAConfig.CertificateAuthorityTLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create ca client cert manager %w", err)
	}

	var caConn *grpc.ClientConn
	var caClient pbCA.CertificateAuthorityClient

	if config.Clients.CAConfig.CertificateAuthorityAddr != "" {
		caConn, err = grpc.Dial(
			config.Clients.CAConfig.CertificateAuthorityAddr,
			grpc.WithTransportCredentials(credentials.NewTLS(caCertManager.GetTLSConfig())),
		)
		if err != nil {
			return nil, fmt.Errorf("cannot connect to certificate authority: %w", err)
		}
		caClient = pbCA.NewCertificateAuthorityClient(caConn)
	}

	manager, err := NewObservationManager()
	if err != nil {
		log.Fatal("unable to initialize new observation manager %w", err)
	}

	oauthCertManager, err := client.New(config.Clients.OAuthProvider.OAuthTLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create oauth client cert manager %w", err)
	}
	auth := kitNetHttp.NewInterceptor(config.Clients.OAuthProvider.JwksURL, oauthCertManager.GetTLSConfig(), authRules, kitNetHttp.RequestMatcher{
		Method: http.MethodGet,
		URI:    regexp.MustCompile(regexp.QuoteMeta(uri.WsStartDevicesObservation) + `.*`),
	}, kitNetHttp.RequestMatcher{
		Method: http.MethodGet,
		URI:    regexp.MustCompile(regexp.QuoteMeta(uri.ClientConfiguration)),
	},
	)
	requestHandler := NewRequestHandler(grpcClient, caClient, &config, manager)

	server := Server{
		server:            NewHTTP(requestHandler, auth),
		cfg:               &config,
		requestHandler:    requestHandler,
		ln:                ln,
		rdConn:            rdConn,
		caConn:            caConn,
		httpCertManager:   httpCertManager,
		rdCertManager:     rdCertManager,
		caCertManager:     caCertManager,
		oauthCertManager:  oauthCertManager,
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
	if s.caConn != nil {
		s.caConn.Close()
	}
	if s.httpCertManager != nil {
		s.httpCertManager.Close()
	}
	if s.rdCertManager != nil {
		s.rdCertManager.Close()
	}
	if s.caCertManager != nil {
		s.caCertManager.Close()
	}
	if s.oauthCertManager != nil {
		s.oauthCertManager.Close()
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
