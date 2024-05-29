package service

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/service"
)

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*service.Service, error) {
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector, "grpc-gateway", fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create open telemetry collector client: %w", err)
	}

	tracerProvider := otelClient.GetTracerProvider()
	validator, err := validator.New(ctx, config.APIs.GRPC.Authorization.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		otelClient.Close()
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	method := "/" + pb.GrpcGateway_ServiceDesc.ServiceName + "/GetHubConfiguration"
	interceptor := server.NewAuth(validator, server.WithWhiteListedMethods(method))
	opts, err := server.MakeDefaultOptions(interceptor, logger, tracerProvider)
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server options: %w", err)
	}
	server, err := server.New(config.APIs.GRPC.Config.BaseConfig, fileWatcher, logger, opts...)
	if err != nil {
		validator.Close()
		otelClient.Close()
		return nil, err
	}
	server.AddCloseFunc(otelClient.Close)
	server.AddCloseFunc(validator.Close)

	closeServerOnError := func(err error) error {
		var errors *multierror.Error
		errors = multierror.Append(errors, err)
		if err2 := server.Close(); err2 != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot close server: %w", err2))
		}
		return errors.ErrorOrNil()
	}
	pool, err := ants.NewPool(config.Clients.Eventbus.GoPoolSize)
	if err != nil {
		return nil, closeServerOnError(fmt.Errorf("cannot create goroutine pool: %w", err))
	}
	server.AddCloseFunc(pool.Release)

	if err := addHandler(server, config, fileWatcher, logger, tracerProvider, pool.Submit); err != nil {
		return nil, closeServerOnError(err)
	}

	return service.New(server), nil
}
