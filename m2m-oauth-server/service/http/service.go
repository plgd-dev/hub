package http

import (
	"fmt"
	"net/http"
	"regexp"

	grpcService "github.com/plgd-dev/hub/v2/m2m-oauth-server/service/grpc"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpJwt "github.com/plgd-dev/hub/v2/pkg/net/http/jwt"
	httpService "github.com/plgd-dev/hub/v2/pkg/net/http/service"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"go.opentelemetry.io/otel/trace"
)

// Service handle HTTP request
type Service struct {
	*httpService.Service
	requestHandler *RequestHandler
}

// New parses configuration and creates new Server with provided store and bus
func New(serviceName string, config Config, m2mOAuthServiceServer *grpcService.M2MOAuthServiceServer, validator *validator.Validator, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Service, error) {
	whiteList := []pkgHttpJwt.RequestMatcher{
		{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.JWKs)),
		},
		{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.OpenIDConfiguration)),
		},
		{
			Method: http.MethodPost,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.Token)),
		},
		{
			Method: http.MethodPost,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.Tokens)),
		},
	}
	service, err := httpService.New(httpService.Config{
		HTTPConnection:    config.Connection,
		HTTPServer:        config.Server,
		ServiceName:       serviceName,
		AuthRules:         pkgHttp.NewDefaultAuthorizationRules(uri.API),
		WhiteEndpointList: whiteList,
		FileWatcher:       fileWatcher,
		Logger:            logger,
		TraceProvider:     tracerProvider,
		Validator:         validator,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create http service: %w", err)
	}

	requestHandler, err := NewRequestHandler(&config, service.GetRouter(), m2mOAuthServiceServer)
	if err != nil {
		_ = service.Close()
		return nil, err
	}

	return &Service{
		Service:        service,
		requestHandler: requestHandler,
	}, nil
}
