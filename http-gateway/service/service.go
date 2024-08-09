package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpJwt "github.com/plgd-dev/hub/v2/pkg/net/http/jwt"
	httpService "github.com/plgd-dev/hub/v2/pkg/net/http/service"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/service"
)

const (
	serviceName                             = "http-gateway"
	AuthorizationWhiteListedEndpointsRegexp = `^\/(a$|[^a].*|ap$|a[^p].*|ap[^i].*|api[^/])`
)

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

	whiteList := []pkgHttpJwt.RequestMatcher{
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
		whiteList = append(whiteList, pkgHttpJwt.RequestMatcher{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(AuthorizationWhiteListedEndpointsRegexp),
		})
	}
	s, err := httpService.New(httpService.Config{
		HTTPConnection:       config.APIs.HTTP.Connection,
		HTTPServer:           config.APIs.HTTP.Server,
		ServiceName:          serviceName,
		AuthRules:            pkgHttp.NewDefaultAuthorizationRules(uri.API),
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
		errC := grpcConn.Close()
		if errC != nil {
			logger.Errorf("error occurs during close connection to grpc-gateway: %v", errC)
		}
	})
	grpcClient := pb.NewGrpcGatewayClient(grpcConn.GRPC())
	client := client.New(grpcClient)
	_, err = NewRequestHandler(&config, s.GetRouter(), client, validator.OpenIDConfiguration(), logger)
	if err != nil {
		var errors *multierror.Error
		errors = multierror.Append(errors, fmt.Errorf("cannot create request handler: %w", err))
		if err2 := s.Close(); err2 != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot close server: %w", err2))
		}
		return nil, errors.ErrorOrNil()
	}

	return service.New(s), nil
}
