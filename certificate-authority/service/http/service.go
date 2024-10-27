package http

import (
	"fmt"
	"net/http"
	"regexp"

	grpcService "github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	"github.com/plgd-dev/hub/v2/certificate-authority/service/uri"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
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
	requestHandler *requestHandler
}

// New parses configuration and creates new Server with provided store and bus
func New(serviceName string, config Config, s store.Store, ca *grpcService.CertificateAuthorityServer, validator *validator.Validator, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Service, error) {
	var whiteList []pkgHttpJwt.RequestMatcher
	if config.CRLEnabled {
		whiteList = append(whiteList, pkgHttpJwt.RequestMatcher{
			Method: http.MethodGet,
			URI:    regexp.MustCompile(regexp.QuoteMeta(uri.SigningRevocationListBase) + `\/.*`),
		})
	}

	service, err := httpService.New(httpService.Config{
		HTTPConnection:       config.Connection,
		HTTPServer:           config.Server,
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
		return nil, fmt.Errorf("cannot create http service: %w", err)
	}

	requestHandler, err := newRequestHandler(&config, service.GetRouter(), ca, s)
	if err != nil {
		_ = service.Close()
		return nil, err
	}

	return &Service{
		Service:        service,
		requestHandler: requestHandler,
	}, nil
}
