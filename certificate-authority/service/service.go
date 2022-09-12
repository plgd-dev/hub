package service

import (
	"context"
	"fmt"

	grpcService "github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	httpService "github.com/plgd-dev/hub/v2/certificate-authority/service/http"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/service"
)

const serviceName = "certificate-authority"

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*service.Service, error) {
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector, "certificate-authority", fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	var closerFn fn.FuncList
	closerFn.AddFunc(otelClient.Close)
	tracerProvider := otelClient.GetTracerProvider()

	ca, err := grpcService.NewCertificateAuthorityServer(config.APIs.GRPC, config.APIs.GRPC.Authorization.OwnerClaim, config.Signer, logger)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}
	httpValidator, err := validator.New(ctx, config.APIs.GRPC.Authorization.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create http validator: %w", err)
	}
	closerFn.AddFunc(httpValidator.Close)
	httpService, err := httpService.New(ctx, serviceName, httpService.Config{
		Connection: listener.Config{
			Addr: config.APIs.HTTP.Addr,
			TLS:  config.APIs.GRPC.TLS,
		},
		Authorization: config.APIs.GRPC.Authorization.Config,
		Server:        config.APIs.HTTP.Server,
	}, ca, httpValidator, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		return nil, fmt.Errorf("cannot create http service: %w", err)
	}
	grpcValidator, err := validator.New(ctx, config.APIs.GRPC.Authorization.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		_ = httpService.Close()
		return nil, fmt.Errorf("cannot create grpc validator: %w", err)
	}
	grpcService, err := grpcService.New(ctx, config.APIs.GRPC, ca, grpcValidator, fileWatcher, logger, tracerProvider)
	if err != nil {
		closerFn.Execute()
		_ = httpService.Close()
		return nil, fmt.Errorf("cannot create grpc validator: %w", err)
	}
	s := service.New(httpService, grpcService)
	s.AddCloseFunc(closerFn.Execute)
	return s, nil
}
