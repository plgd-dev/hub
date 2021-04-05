package service

import (
	"context"
	"crypto/tls"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"

	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	"github.com/plgd-dev/kit/security/jwt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func NewService(config Config, logger *zap.Logger) (*kitNetGrpc.Server, error) {
	log.Info(config.String())

	var oauthCertManager *client.CertManager = nil
	var oauthTLSConfig *tls.Config = nil
	err := config.Clients.OAuthProvider.TLSConfig.Validate()
	if err != nil {
		log.Errorf("failed to validate client tls config: %v", err)
	} else {
		oauthCertManager, err := client.New(config.Clients.OAuthProvider.TLSConfig, logger)
		if err != nil {
			log.Errorf("cannot create oauth client cert manager %v", err)
		} else {
			oauthTLSConfig = oauthCertManager.GetTLSConfig()
		}
	}
	auth := NewAuth(config.Clients.OAuthProvider.JwksURL, config.Clients.OAuthProvider.OwnerClaim, oauthTLSConfig)

	var streamInterceptors []grpc.StreamServerInterceptor
	if logger.Core().Enabled(zapcore.DebugLevel) {
		streamInterceptors = append(streamInterceptors, grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.StreamServerInterceptor(logger))
	}
	streamInterceptors = append(streamInterceptors, auth.Stream())

	var unaryInterceptors []grpc.UnaryServerInterceptor
	if logger.Core().Enabled(zapcore.DebugLevel) {
		unaryInterceptors = append(unaryInterceptors, grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(logger))
	}
	unaryInterceptors = append(unaryInterceptors, auth.Unary())

	grpcCertManager, err := server.New(config.Service.Grpc.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create grpc server cert manager %w", err)
	}
	server, err := kitNetGrpc.NewServer(config.Service.Grpc.Addr, grpc.Creds(credentials.NewTLS(grpcCertManager.GetTLSConfig())),
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
	server.AddCloseFunc(func() {
		if oauthCertManager != nil { oauthCertManager.Close() }
		grpcCertManager.Close()
	})

	if err := AddHandler(server, config, logger); err != nil {
		return nil, err
	}

	return server, nil
}

func makeAuthFunc(jwksUrl, ownerClaim string, tls *tls.Config) func(ctx context.Context, method string) (context.Context, error) {
	interceptor := kitNetGrpc.ValidateJWT(jwksUrl, tls, func(ctx context.Context, method string) kitNetGrpc.Claims {
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

func NewAuth(jwksUrl, ownerClaim string, tls *tls.Config) kitNetGrpc.AuthInterceptors {
	return kitNetGrpc.MakeAuthInterceptors(makeAuthFunc(jwksUrl, ownerClaim, tls))
}