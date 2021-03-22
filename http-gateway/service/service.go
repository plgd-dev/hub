package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/panjf2000/ants/v2"
	pbCA "github.com/plgd-dev/cloud/certificate-authority/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	raClient "github.com/plgd-dev/cloud/resource-aggregate/client"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/kit/log"
	kitNetHttp "github.com/plgd-dev/kit/net/http"
	"github.com/plgd-dev/kit/security/certManager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func logError(err error) { log.Error(err) }

//Server handle HTTP request
type Server struct {
	server             *http.Server
	cfg                *Config
	requestHandler     *RequestHandler
	ln                 net.Listener
	listenCertManager  certManager.CertManager
	rdConn             *grpc.ClientConn
	caConn             *grpc.ClientConn
	raConn             *grpc.ClientConn
	resourceSubscriber eventbus.Subscriber
}

func buildWhiteList(uidirectory string, whiteList *[]kitNetHttp.RequestMatcher) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		path = strings.TrimLeft(path, uidirectory)
		log.Debugf("white listed path: %v", path)
		*whiteList = append(*whiteList, kitNetHttp.RequestMatcher{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(regexp.QuoteMeta(path)),
		})
		return nil
	}
}

// New parses configuration and creates new Server with provided store and bus
func New(cfg Config) (*Server, error) {
	cfg = cfg.checkForDefaults()
	log.Info(cfg.String())

	listenCertManager, err := certManager.NewCertManager(cfg.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create listen cert manager: %v", err)
	}
	dialCertManager, err := certManager.NewCertManager(cfg.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create dial cert manager: %v", err)
	}
	listenTLSCfg := listenCertManager.GetServerTLSConfig()

	ln, err := tls.Listen("tcp", cfg.Address, listenTLSCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot listen tls and serve: %v", err)
	}

	rdConn, err := grpc.Dial(
		cfg.ResourceDirectoryAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialCertManager.GetClientTLSConfig())),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource directory: %w", err)
	}
	resourceDirectoryClient := pb.NewGrpcGatewayClient(rdConn)
	client := client.New(resourceDirectoryClient)

	pool, err := ants.NewPool(cfg.GoRoutinePoolSize)
	if err != nil {
		return nil, fmt.Errorf("cannot create goroutine pool: %w", err)
	}

	resourceSubscriber, err := nats.NewSubscriber(cfg.Nats, pool.Submit, func(err error) { log.Errorf("error occurs during receiving event: %v", err) }, nats.WithTLS(dialCertManager.GetClientTLSConfig()))
	if err != nil {
		return nil, fmt.Errorf("cannot create eventbus subscriber: %w", err)
	}

	raConn, err := grpc.Dial(
		cfg.ResourceAggregateAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(dialCertManager.GetClientTLSConfig())),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	resourceAggregateClient := raClient.New(raConn, resourceSubscriber)

	var caConn *grpc.ClientConn
	var caClient pbCA.CertificateAuthorityClient
	if cfg.CertificateAuthorityAddr != "" {
		caConn, err = grpc.Dial(
			cfg.CertificateAuthorityAddr,
			grpc.WithTransportCredentials(credentials.NewTLS(dialCertManager.GetClientTLSConfig())),
		)
		if err != nil {
			return nil, fmt.Errorf("cannot connect to certificate authority: %w", err)
		}
		caClient = pbCA.NewCertificateAuthorityClient(caConn)
	}

	manager, err := NewObservationManager()
	if err != nil {
		return nil, fmt.Errorf("unable to initialize new observation manager %v", err)
	}

	whiteList := []kitNetHttp.RequestMatcher{
		{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.WsStartDevicesObservation) + `.*`),
		},
		{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.ClientConfiguration)),
		},
	}
	if cfg.UI.Enabled {
		whiteList = append(whiteList, kitNetHttp.RequestMatcher{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(`(\/[^a]pi\/.*)|(\/a[^p]i\/.*)|(\/ap[^i]\/.*)||(\/api[^/].*)`),
		})
	}
	auth := kitNetHttp.NewInterceptor(cfg.JwksURL, dialCertManager.GetClientTLSConfig(), authRules, whiteList...)
	requestHandler := NewRequestHandler(client, caClient, &cfg, manager, resourceAggregateClient)

	server := Server{
		server:             NewHTTP(requestHandler, auth),
		cfg:                &cfg,
		requestHandler:     requestHandler,
		ln:                 ln,
		listenCertManager:  listenCertManager,
		rdConn:             rdConn,
		caConn:             caConn,
		raConn:             raConn,
		resourceSubscriber: resourceSubscriber,
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
	s.raConn.Close()
	if s.caConn != nil {
		s.caConn.Close()
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
