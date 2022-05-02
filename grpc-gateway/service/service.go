package service

import (
	"context"
	"fmt"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	*server.Server
}

func New(ctx context.Context, config Config, logger log.Logger) (*Service, error) {
	tracerProvider := trace.NewNoopTracerProvider()
	validator, err := validator.New(ctx, config.APIs.GRPC.Authorization.Config, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	method := "/" + pb.GrpcGateway_ServiceDesc.ServiceName + "/GetHubConfiguration"
	interceptor := server.NewAuth(validator, server.WithWhiteListedMethods(method))
	opts, err := server.MakeDefaultOptions(interceptor, logger, tracerProvider)
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server options: %w", err)
	}
	server, err := server.New(config.APIs.GRPC.Config, logger, opts...)

	if err != nil {
		validator.Close()
		return nil, err
	}
	server.AddCloseFunc(validator.Close)

	pool, err := ants.NewPool(config.Clients.Eventbus.GoPoolSize)
	if err != nil {
		server.Close()
		return nil, fmt.Errorf("cannot create goroutine pool: %w", err)
	}
	server.AddCloseFunc(pool.Release)

	if err := AddHandler(ctx, server, config, logger, tracerProvider, pool.Submit); err != nil {
		server.Close()
		return nil, err
	}

	return &Service{
		Server: server,
	}, nil
}
