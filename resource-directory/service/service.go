package service

import (
	"context"
	"fmt"

	"github.com/panjf2000/ants/v2"

	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/pkg/security/jwt"
	"github.com/plgd-dev/cloud/pkg/security/jwt/validator"
)

type Service struct {
	*server.Server
}

func New(ctx context.Context, config Config, logger log.Logger) (*Service, error) {
	validator, err := validator.New(ctx, config.APIs.GRPC.Authorization, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	opts, err := server.MakeDefaultOptions(NewAuth(validator, config.Clients.AuthServer.OwnerClaim), logger)
	if err != nil {
		validator.Close()
		return nil, fmt.Errorf("cannot create grpc server options: %w", err)
	}
	server, err := server.New(config.APIs.GRPC, logger, opts...)

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

	if err := AddHandler(ctx, server, config.Clients, config.ExposedCloudConfiguration, logger, pool.Submit); err != nil {
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
		case "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetCloudConfiguration":
			return ctx, nil
		}
		token, _ := kitNetGrpc.TokenFromMD(ctx)
		ctx, err := interceptor(ctx, method)
		if err != nil {
			log.Errorf("auth interceptor %v %v: %w", method, token, err)
			return ctx, err
		}
		owner, err := kitNetGrpc.OwnerFromMD(ctx)
		if err != nil {
			owner, err = kitNetGrpc.OwnerFromTokenMD(ctx, ownerClaim)
			if err == nil {
				ctx = kitNetGrpc.CtxWithIncomingOwner(ctx, owner)
			}
		}
		if err != nil {
			log.Errorf("auth cannot get owner: %v", err)
			return ctx, err
		}
		return kitNetGrpc.CtxWithOwner(ctx, owner), nil
	}
}

func NewAuth(validator kitNetGrpc.Validator, ownerClaim string) kitNetGrpc.AuthInterceptors {
	return kitNetGrpc.MakeAuthInterceptors(makeAuthFunc(validator, ownerClaim))
}
