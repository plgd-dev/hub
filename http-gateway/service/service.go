package service

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"regexp"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	grpcService "github.com/go-ocf/cloud/grpc-gateway/service"
	"github.com/go-ocf/cloud/http-gateway/uri"
	"github.com/go-ocf/kit/log"
	kitNetHttp "github.com/go-ocf/kit/net/http"
	"github.com/go-ocf/kit/security/certManager"
)

func logError(err error) { log.Error(err) }

//Server handle HTTP request
type Server struct {
	server            *http.Server
	cfg               *Config
	requestHandler    *RequestHandler
	ln                net.Listener
	listenCertManager certManager.CertManager
	ch                *inprocgrpc.Channel
}

// New parses configuration and creates new Server with provided store and bus
func New(config string) (*Server, error) {
	cfg, err := ParseConfig(config)
	if err != nil {
		return nil, err
	}

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

	grpcServerHandler, err := grpcService.NewRequestHandlerFromConfig(cfg.HandlerConfig, dialCertManager.GetClientTLSConfig())
	if err != nil {
		log.Fatalf("cannot listen tls and serve: %w", err)
	}

	var ch inprocgrpc.Channel
	pb.RegisterHandlerGrpcGateway(&ch, grpcServerHandler)
	grpcClient := pb.NewGrpcGatewayChannelClient(&ch)

	client, err := client.NewClient(cfg.AccessTokenURL, grpcClient)
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
		ch:                &ch,
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
	if s.listenCertManager != nil {
		s.listenCertManager.Close()
	}
	return s.server.Shutdown(context.Background())
}

var authRules = map[string][]kitNetHttp.AuthArgs{
	http.MethodGet: {
		{
			URI: regexp.MustCompile(regexp.QuoteMeta(uri.API) + `\/.*`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`openid`),
			},
		},
	},
	http.MethodPost: {
		{
			URI: regexp.MustCompile(regexp.QuoteMeta(uri.API) + `\/.*`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`openid`),
			},
		},
	},
	http.MethodDelete: {
		{
			URI: regexp.MustCompile(regexp.QuoteMeta(uri.API) + `\/.*`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`openid`),
			},
		},
	},
	http.MethodPut: {
		{
			URI: regexp.MustCompile(regexp.QuoteMeta(uri.API) + `\/.*`),
			Scopes: []*regexp.Regexp{
				regexp.MustCompile(`openid`),
			},
		},
	},
}
