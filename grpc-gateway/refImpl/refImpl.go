package refImpl

import (
	"context"
	"crypto/tls"
	"fmt"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/grpc-gateway/service"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	"github.com/plgd-dev/kit/security/jwt"
)

func Init(config service.Config) (*kitNetGrpc.Server, error) {
	logger, err := log.NewLogger(config.Log)
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)
	log.Info(config.String())

	oauthCertManager, err := client.New(config.Clients.OAuthProvider.OAuthTLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create oauth client cert manager %w", err)
	}
	auth := NewAuth(config.Clients.OAuthProvider.JwksURL, oauthCertManager.GetTLSConfig())
	var streamInterceptors []grpc.StreamServerInterceptor
	if config.Log.Debug {
		streamInterceptors = append(streamInterceptors, grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.StreamServerInterceptor(logger))
	}
	streamInterceptors = append(streamInterceptors, auth.Stream())

	var unaryInterceptors []grpc.UnaryServerInterceptor
	if config.Log.Debug {
		unaryInterceptors = append(unaryInterceptors, grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(logger))
	}
	unaryInterceptors = append(unaryInterceptors, auth.Unary())

	grpcCertManager, err := server.New(config.Service.GrpcConfig.TLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create grpc server cert manager %w", err)
	}
	server, err := kitNetGrpc.NewServer(config.Service.GrpcConfig.Addr, grpc.Creds(credentials.NewTLS(grpcCertManager.GetTLSConfig())),
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

	rdCertManager, err := client.New(config.Clients.RDConfig.ResourceDirectoryTLSConfig, logger)
	if err != nil {
		log.Errorf("cannot create rd client cert manager %w", err)
	}
	if err := service.AddHandler(server, config.Clients.RDConfig, rdCertManager.GetTLSConfig()); err != nil {
		return nil, err
	}

	server.AddCloseFunc(func() {
		oauthCertManager.Close()
		grpcCertManager.Close()
		rdCertManager.Close()
	})

	return server, nil
}

func NewAuth(jwksUrl string, tls *tls.Config) kitNetGrpc.AuthInterceptors {
	interceptor := kitNetGrpc.ValidateJWT(jwksUrl, tls, func(ctx context.Context, method string) kitNetGrpc.Claims {
		return jwt.NewScopeClaims()
	})
	return kitNetGrpc.MakeAuthInterceptors(func(ctx context.Context, method string) (context.Context, error) {
		ctx, err := interceptor(ctx, method)
		if err != nil {
			log.Errorf("auth interceptor %v %v: %v", method, err)
			return ctx, err
		}
		token, err := kitNetGrpc.TokenFromMD(ctx)
		if err != nil {
			log.Errorf("auth cannot get token: %v", err)
			return ctx, err
		}
		return kitNetGrpc.CtxWithToken(ctx, token), nil
	}, "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetClientConfiguration")
}
