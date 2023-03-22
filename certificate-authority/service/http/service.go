package http

import (
	"fmt"
	"net/http"
	"regexp"

	grpcService "github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	httpService "github.com/plgd-dev/hub/v2/pkg/net/http/service"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"go.opentelemetry.io/otel/trace"
)

// Service handle HTTP request
type Service struct {
	*httpService.Service
	requestHandler *RequestHandler
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

// New parses configuration and creates new Server with provided store and bus
func New(serviceName string, config Config, ca *grpcService.CertificateAuthorityServer, validator *validator.Validator, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Service, error) {
	service, err := httpService.New(httpService.Config{
		HTTPConnection: config.Connection,
		HTTPServer:     config.Server,
		ServiceName:    serviceName,
		AuthRules:      authRules,
		// WhiteEndpointList:      whiteList,
		FileWatcher:   fileWatcher,
		Logger:        logger,
		TraceProvider: tracerProvider,
		Validator:     validator,
		// QueryCaseInsensitive: map[string]string{},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create http service: %w", err)
	}

	requestHandler, err := NewRequestHandler(&config, service.GetRouter(), ca)
	if err != nil {
		_ = service.Close()
		return nil, err
	}

	return &Service{
		Service:        service,
		requestHandler: requestHandler,
	}, nil
}
