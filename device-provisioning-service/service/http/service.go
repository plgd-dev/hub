package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service/grpc"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpJwt "github.com/plgd-dev/hub/v2/pkg/net/http/jwt"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	httpServer *http.Server
	listener   *listener.Server
}

// New creates new HTTP service
func New(ctx context.Context, serviceName string, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, store store.Store) (*Service, error) {
	validator, err := validator.New(ctx, config.Authorization.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	listener, err := listener.New(config.Connection, fileWatcher, logger, tracerProvider)
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server: %w", err)
	}
	listener.AddCloseFunc(validator.Close)

	ch := new(inprocgrpc.Channel)
	pb.RegisterDeviceProvisionServiceServer(ch, grpc.NewDeviceProvisionServiceServer(store, config.Authorization.OwnerClaim))
	grpcClient := pb.NewDeviceProvisionServiceClient(ch)

	auth := pkgHttpJwt.NewInterceptorWithValidator(validator, pkgHttp.NewDefaultAuthorizationRules(APIV1))
	mux := serverMux.New()
	r := serverMux.NewRouter(queryCaseInsensitive, auth)

	// register grpc-proxy handler
	if err := pb.RegisterDeviceProvisionServiceHandlerClient(context.Background(), mux, grpcClient); err != nil {
		return nil, fmt.Errorf("failed to register grpc-gateway handler: %w", err)
	}
	r.PathPrefix("/").Handler(mux)
	httpServer := &http.Server{
		Handler:           pkgHttp.OpenTelemetryNewHandler(r, serviceName, tracerProvider),
		ReadTimeout:       config.Server.ReadTimeout,
		ReadHeaderTimeout: config.Server.ReadHeaderTimeout,
		WriteTimeout:      config.Server.WriteTimeout,
		IdleTimeout:       config.Server.IdleTimeout,
	}

	return &Service{
		httpServer: httpServer,
		listener:   listener,
	}, nil
}

// Serve starts the service's HTTP server and blocks
func (s *Service) Serve() error {
	return s.httpServer.Serve(s.listener)
}

// Shutdown ends serving
func (s *Service) Close() error {
	return s.httpServer.Shutdown(context.Background())
}

const (
	IDFilterQueryKey                = "idFilter"
	EnrollmentGroupIDFilterQueryKey = "enrollmentGroupIdFilter"
	DeviceIDFilterQueryKey          = "deviceIdFilter"
)

var queryCaseInsensitive = map[string]string{
	strings.ToLower(IDFilterQueryKey):                IDFilterQueryKey,
	strings.ToLower(EnrollmentGroupIDFilterQueryKey): EnrollmentGroupIDFilterQueryKey,
	strings.ToLower(DeviceIDFilterQueryKey):          DeviceIDFilterQueryKey,
}
