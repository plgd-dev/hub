package service

import (
	"context"
	"fmt"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
)

type Service struct {
	*server.Server
}

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger) (*Service, error) {
	otelClient, err := otelClient.New(ctx, config.Clients.OpenTelemetryCollector, "resource-directory", fileWatcher, logger)
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
		otelClient.Close()
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server options: %w", err)
	}
	server, err := server.New(config.APIs.GRPC.Config, fileWatcher, logger, opts...)
	if err != nil {
		otelClient.Close()
		validator.Close()
		return nil, err
	}
	server.AddCloseFunc(otelClient.Close)
	server.AddCloseFunc(validator.Close)

	pool, err := ants.NewPool(config.Clients.Eventbus.GoPoolSize)
	if err != nil {
		err = fmt.Errorf("cannot create goroutine pool: %w", err)
		err2 := server.Close()
		if err2 != nil {
			err = fmt.Errorf(`[%w, "cannot close server: %v"]`, err, err2)
		}
		return nil, err
	}
	server.AddCloseFunc(pool.Release)

	if err := AddHandler(ctx, server, config, config.ExposedHubConfiguration, fileWatcher, logger, tracerProvider, pool.Submit); err != nil {
		err2 := server.Close()
		if err2 != nil {
			err = fmt.Errorf(`[%w, "cannot close server: %v"]`, err, err2)
		}
		return nil, err
	}

	return &Service{
		Server: server,
	}, nil
}

func makeAuthFunc(validator kitNetGrpc.Validator) func(ctx context.Context, method string) (context.Context, error) {
	interceptor := kitNetGrpc.ValidateJWTWithValidator(validator, func(ctx context.Context, method string) kitNetGrpc.Claims {
		return jwt.NewScopeClaims()
	})
	return func(ctx context.Context, method string) (context.Context, error) {
		if method == "/"+pb.GrpcGateway_ServiceDesc.ServiceName+"/GetHubConfiguration" {
			return ctx, nil
		}
		token, _ := kitNetGrpc.TokenFromMD(ctx)
		ctx, err := interceptor(ctx, method)
		if err != nil {
			log.Errorf("auth interceptor %v %v: %w", method, token, err)
			return ctx, err
		}
		return ctx, err
	}
}

func NewAuth(validator kitNetGrpc.Validator) kitNetGrpc.AuthInterceptors {
	return kitNetGrpc.MakeAuthInterceptors(makeAuthFunc(validator))
}
