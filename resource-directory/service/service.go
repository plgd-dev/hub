package service

import (
	"context"
	"fmt"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"github.com/plgd-dev/cloud/v2/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/v2/pkg/security/jwt"
	"github.com/plgd-dev/cloud/v2/pkg/security/jwt/validator"
)

type Service struct {
	*server.Server
}

func New(ctx context.Context, config Config, logger log.Logger) (*Service, error) {
	validator, err := validator.New(ctx, config.APIs.GRPC.Authorization.Config, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	method := "/" + pb.GrpcGateway_ServiceDesc.ServiceName + "/GetCloudConfiguration"
	interceptor := server.NewAuth(validator, server.WithWhiteListedMethods(method))
	opts, err := server.MakeDefaultOptions(interceptor, logger)
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

	if err := AddHandler(ctx, server, config, config.ExposedCloudConfiguration, logger, pool.Submit); err != nil {
		server.Close()
		return nil, err
	}

	return &Service{
		Server: server,
	}, nil
}

func makeAuthFunc(validator kitNetGrpc.Validator, ownerClaim string) func(ctx context.Context, method string) (context.Context, error) {
	interceptor := kitNetGrpc.ValidateJWTWithValidator(validator, func(ctx context.Context, method string) kitNetGrpc.Claims {
		return jwt.NewScopeClaims()
	})
	return func(ctx context.Context, method string) (context.Context, error) {
		switch method {
		case "/" + pb.GrpcGateway_ServiceDesc.ServiceName + "/GetCloudConfiguration":
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

func NewAuth(validator kitNetGrpc.Validator, ownerClaim string) kitNetGrpc.AuthInterceptors {
	return kitNetGrpc.MakeAuthInterceptors(makeAuthFunc(validator, ownerClaim))
}
