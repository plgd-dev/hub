package service

import (
	"context"
	"fmt"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"google.golang.org/grpc"

	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/pkg/security/jwt"
	"github.com/plgd-dev/cloud/pkg/security/jwt/validator"
)

type Service struct {
	*server.Server
}

func New(ctx context.Context, config Config, logger *zap.Logger) (*Service, error) {
	validator, err := validator.New(config.Clients.OAuthProvider.Jwks, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create validator: %w", err)
	}
	auth := NewAuth(validator, config.Clients.OAuthProvider.OwnerClaim)
	streamInterceptors := []grpc.StreamServerInterceptor{}
	if logger.Core().Enabled(zapcore.DebugLevel) {
		streamInterceptors = append(streamInterceptors, grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.StreamServerInterceptor(logger))
	}
	streamInterceptors = append(streamInterceptors, auth.Stream())

	unaryInterceptors := []grpc.UnaryServerInterceptor{}
	if logger.Core().Enabled(zapcore.DebugLevel) {
		unaryInterceptors = append(unaryInterceptors, grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(logger))
	}
	unaryInterceptors = append(unaryInterceptors, auth.Unary())

	server, err := server.New(config.APIs.GRPC.Server, logger,
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			streamInterceptors...,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryInterceptors...,
		)),
	)

	if err != nil {
		return nil, err
	}
	server.AddCloseFunc(validator.Close)

	pool, err := ants.NewPool(config.APIs.GRPC.Capabilities.GoRoutinePoolSize)
	if err != nil {
		return nil, fmt.Errorf("cannot create goroutine pool: %w", err)
	}
	server.AddCloseFunc(pool.Release)

	if err := AddHandler(server, config.Clients, logger, pool.Submit); err != nil {
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
		case "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetClientConfiguration":
			return ctx, nil
		}
		token, _ := kitNetGrpc.TokenFromMD(ctx)
		ctx, err := interceptor(ctx, method)
		if err != nil {
			log.Errorf("auth interceptor %v %v: %v", method, token, err)
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
