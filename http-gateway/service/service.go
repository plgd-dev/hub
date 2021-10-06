package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/plgd-dev/cloud/v2/pkg/log"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/client"
	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/v2/http-gateway/uri"
	grpcClient "github.com/plgd-dev/cloud/v2/pkg/net/grpc/client"
	kitNetHttp "github.com/plgd-dev/cloud/v2/pkg/net/http"
	"github.com/plgd-dev/cloud/v2/pkg/net/listener"
	"github.com/plgd-dev/cloud/v2/pkg/security/jwt/validator"
)

//Server handle HTTP request
type Server struct {
	server         *http.Server
	config         *Config
	requestHandler *RequestHandler
	listener       *listener.Server
}

// New parses configuration and creates new Server with provided store and bus
func New(ctx context.Context, config Config, logger log.Logger) (*Server, error) {
	validator, err := validator.New(ctx, config.APIs.HTTP.Authorization, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}

	listener, err := listener.New(config.APIs.HTTP.Connection, logger)
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server: %w", err)
	}
	listener.AddCloseFunc(validator.Close)

	grpcConn, err := grpcClient.New(config.Clients.GrpcGateway.Connection, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource directory: %w", err)
	}
	listener.AddCloseFunc(func() {
		err := grpcConn.Close()
		if err != nil {
			logger.Errorf("error occurs during close connection to resource-directory: %v", err)
		}
	})
	grpcClient := pb.NewGrpcGatewayClient(grpcConn.GRPC())
	client := client.New(grpcClient)

	whiteList := []kitNetHttp.RequestMatcher{
		{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.APIWS) + `.*`),
		},
		{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(regexp.QuoteMeta("/api/v1/clientConfiguration")),
		},
		{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.CloudConfiguration)),
		},
	}
	if config.UI.Enabled {
		whiteList = append(whiteList, kitNetHttp.RequestMatcher{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(`(\/[^a]pi\/.*)|(\/a[^p]i\/.*)|(\/ap[^i]\/.*)||(\/api[^/].*)`),
		})
	}
	auth := kitNetHttp.NewInterceptorWithValidator(validator, authRules, whiteList...)
	requestHandler := NewRequestHandler(&config, client)

	http, err := NewHTTP(requestHandler, auth)
	if err != nil {
		return nil, fmt.Errorf("cannot create http server: %w", err)
	}

	server := Server{
		server:         http,
		config:         &config,
		requestHandler: requestHandler,
		listener:       listener,
	}

	return &server, nil
}

// Serve starts the service's HTTP server and blocks
func (s *Server) Serve() error {
	return s.server.Serve(s.listener)
}

// Shutdown ends serving
func (s *Server) Shutdown() error {
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
