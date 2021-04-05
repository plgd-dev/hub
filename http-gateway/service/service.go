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
	grpcClient "github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetHttp "github.com/plgd-dev/cloud/pkg/net/http"
	raClient "github.com/plgd-dev/cloud/resource-aggregate/client"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/kit/log"
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
	listenCertManager *server.CertManager
	rdConn            *grpc.ClientConn
	caConn            *grpc.ClientConn
	raConn            *grpc.ClientConn
	resourceSubscriber eventbus.Subscriber

	rdCertManager     *client.CertManager
	caCertManager     *client.CertManager
	raCertManager     *client.CertManager
	natsCertManager   *client.CertManager
	oauthCertManager  *client.CertManager
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
func New(config Config) (*Server, error) {
	logger, err := log.NewLogger(config.Log)
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)
	log.Info(config.String())

	config = config.checkForDefaults()
	listenCertManager, err := server.New(config.Service.Http.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create http server cert manager %w", err)
	}
	ln, err := tls.Listen("tcp", config.Service.Http.Addr, listenCertManager.GetTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("cannot listen tls and serve: %v", err)
	}

	rdCertManager, err := client.New(config.Clients.ResourceDirectory.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create rd client cert manager %w", err)
	}
	rdConn, err := grpc.Dial(
		config.Clients.ResourceDirectory.Addr,
		grpc.WithTransportCredentials(credentials.NewTLS(rdCertManager.GetTLSConfig())),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource directory: %w", err)
	}
	resourceDirectoryClient := pb.NewGrpcGatewayClient(rdConn)
	grpcGwClient := grpcClient.New(resourceDirectoryClient)

	natsCertManager, err := client.New(config.Clients.Nats.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create nats client cert manager %w", err)
	}
	pool, err := ants.NewPool(config.Clients.GoRoutinePoolSize)
	if err != nil {
		return nil, fmt.Errorf("cannot create goroutine pool: %w", err)
	}
	resourceSubscriber, err := nats.NewSubscriber(config.Clients.Nats, pool.Submit, func(err error) { log.Errorf("error occurs during receiving event: %v", err) }, nats.WithTLS(natsCertManager.GetTLSConfig()))
	if err != nil {
		return nil, fmt.Errorf("cannot create eventbus subscriber: %w", err)
	}

	raCertManager, err := client.New(config.Clients.ResourceAggregate.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create ca client cert manager %w", err)
	}
	raConn, err := grpc.Dial(
		config.Clients.ResourceAggregate.Addr,
		grpc.WithTransportCredentials(credentials.NewTLS(raCertManager.GetTLSConfig())),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	resourceAggregateClient := raClient.New(raConn, resourceSubscriber)

	caCertManager, err := client.New(config.Clients.CertificateAuthority.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create ca client cert manager %w", err)
	}
	var caConn *grpc.ClientConn
	var caClient pbCA.CertificateAuthorityClient
	if config.Clients.CertificateAuthority.Addr != "" {

		caConn, err = grpc.Dial(
			config.Clients.CertificateAuthority.Addr,
			grpc.WithTransportCredentials(credentials.NewTLS(caCertManager.GetTLSConfig())),
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

	var oauthCertManager *client.CertManager = nil
	var oauthTLSConfig *tls.Config = nil
	err = config.Clients.OAuthProvider.TLSConfig.Validate()
	if err != nil {
		log.Errorf("failed to validate client tls config: %v", err)
	} else {
		oauthCertManager, err := client.New(config.Clients.OAuthProvider.TLSConfig, logger)
		if err != nil {
			log.Errorf("cannot create oauth client cert manager %v", err)
		} else {
			oauthTLSConfig = oauthCertManager.GetTLSConfig()
		}
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
	if config.UI.Enabled {
		whiteList = append(whiteList, kitNetHttp.RequestMatcher{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(`(\/[^a]pi\/.*)|(\/a[^p]i\/.*)|(\/ap[^i]\/.*)||(\/api[^/].*)`),
		})
	}

	auth := kitNetHttp.NewInterceptor(config.Clients.OAuthProvider.JwksURL, oauthTLSConfig, authRules, whiteList...)
	requestHandler := NewRequestHandler(grpcGwClient, caClient, &config, manager, resourceAggregateClient)

	server := Server{
		server:            NewHTTP(requestHandler, auth),
		cfg:               &config,
		requestHandler:    requestHandler,
		ln:                ln,
		listenCertManager:  listenCertManager,
		rdConn:            rdConn,
		caConn:            caConn,
		raConn:             raConn,
		resourceSubscriber: resourceSubscriber,
		rdCertManager:     rdCertManager,
		caCertManager:     caCertManager,
		raCertManager:     raCertManager,
		natsCertManager:   natsCertManager,
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
	s.raConn.Close()
	if s.caConn != nil {
		s.caConn.Close()
	}
	if s.listenCertManager != nil {
		s.listenCertManager.Close()
	}
	if s.rdCertManager != nil {
		s.rdCertManager.Close()
	}
	if s.caCertManager != nil {
		s.caCertManager.Close()
	}
	if s.raCertManager != nil {
		s.raCertManager.Close()
	}
	if s.natsCertManager != nil {
		s.natsCertManager.Close()
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
