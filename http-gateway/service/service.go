package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	kitNetHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	httpService "github.com/plgd-dev/hub/v2/pkg/net/http/service"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/service"
)

const (
	serviceName                             = "http-gateway"
	AuthorizationWhiteListedEndpointsRegexp = `^\/(a$|[^a].*|ap$|a[^p].*|ap[^i].*|api[^/])`
)

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

// New parses configuration and creates new Server with provided store and bus
func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*service.Service, error) {
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector.Config, serviceName, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	tracerProvider := otelClient.GetTracerProvider()
	validator, err := validator.New(ctx, config.APIs.HTTP.Authorization, fileWatcher, logger, tracerProvider)
	if err != nil {
		otelClient.Close()
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}

	whiteList := []kitNetHttp.RequestMatcher{
		{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.APIWS) + `.*`),
		},
		{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.HubConfiguration)),
		},
	}
	if config.UI.Enabled {
		whiteList = append(whiteList, kitNetHttp.RequestMatcher{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(AuthorizationWhiteListedEndpointsRegexp),
		})
	}
	s, err := httpService.New(ctx, httpService.Config{
		HTTPConnection:       config.APIs.HTTP.Connection,
		HTTPServer:           config.APIs.HTTP.Server,
		ServiceName:          serviceName,
		AuthRules:            authRules,
		WhiteEndpointList:    whiteList,
		FileWatcher:          fileWatcher,
		Logger:               logger,
		TraceProvider:        tracerProvider,
		Validator:            validator,
		QueryCaseInsensitive: uri.QueryCaseInsensitive,
	})
	if err != nil {
		otelClient.Close()
		validator.Close()
		return nil, fmt.Errorf("cannot create http service: %w", err)
	}
	s.AddCloseFunc(otelClient.Close)
	s.AddCloseFunc(validator.Close)

	grpcConn, err := grpcClient.New(config.Clients.GrpcGateway.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to grpc-gateway: %w", err)
	}
	s.AddCloseFunc(func() {
		err := grpcConn.Close()
		if err != nil {
			logger.Errorf("error occurs during close connection to grpc-gateway: %v", err)
		}
	})
	grpcClient := pb.NewGrpcGatewayClient(grpcConn.GRPC())
	client := client.New(grpcClient)
	_, err = NewRequestHandler(&config, s.GetRouter(), client)
	if err != nil {
		err = fmt.Errorf("cannot create request handler: %w", err)
		err2 := s.Close()
		if err2 != nil {
			err = fmt.Errorf(`[%w, "cannot close server: %v"]`, err, err2)
		}
		return nil, err
	}

	return service.New(s), nil
}
