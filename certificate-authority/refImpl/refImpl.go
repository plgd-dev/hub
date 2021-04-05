package refImpl

import (
	"context"
	"crypto/tls"
	"fmt"
	"go.uber.org/zap"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"

	"github.com/plgd-dev/cloud/certificate-authority/pb"
	"github.com/plgd-dev/cloud/certificate-authority/service"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/security/jwt"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type RefImpl struct {
	handle            *service.RequestHandler
	server            *kitNetGrpc.Server
	oauthCertManager  *client.CertManager
	grpcCertManager   *server.CertManager

}

func Init(config service.Config) (*RefImpl, error) {
	logger, err := log.NewLogger(config.Log)
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)
	log.Info(config.String())

	var oauthCertManager *client.CertManager = nil
	var oauthTLSConfig *tls.Config = nil
	err = config.Clients.OAuthProvider.TLSConfig.Validate()
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

	auth := NewAuth(config.Clients.OAuthProvider.JwksURL, oauthTLSConfig, "openid")
	r, err := InitWithAuth(config, auth, logger)
	if err != nil {
		if oauthCertManager !=nil { oauthCertManager.Close() }
		return nil, err
	}
	r.oauthCertManager = oauthCertManager
	return r, nil
}

// NewRequestHandlerFromConfig creates RegisterGrpcGatewayServer with all dependencies.
func NewRefImplFromConfig(config service.Config, auth kitNetGrpc.AuthInterceptors, logger *zap.Logger) (*RefImpl, error) {

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

	grpcCertManager, err := server.New(config.Service.Grpc.TLSConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create server cert manager %w", err)
	}
	svr, err := kitNetGrpc.NewServer(config.Service.Grpc.Addr, grpc.Creds(credentials.NewTLS(grpcCertManager.GetTLSConfig())),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			streamInterceptors...,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryInterceptors...,
		)),
	)
	if err != nil {
		grpcCertManager.Close()
		return nil, err
	}

	handler, err := service.NewRequestHandlerFromConfig(config.Clients.Signer)
	if err != nil {
		return nil, err
	}
	return &RefImpl{
		handle:            handler,
		grpcCertManager: grpcCertManager,
		server:            svr,
	}, nil
}

func InitWithAuth(config service.Config, auth kitNetGrpc.AuthInterceptors, logger *zap.Logger) (*RefImpl, error) {

	impl, err := NewRefImplFromConfig(config, auth, logger)
	if err != nil {
		return nil, err
	}

	pb.RegisterCertificateAuthorityServer(impl.server.Server, impl.handle)

	return impl, nil
}

func (r *RefImpl) Serve() error {
	return r.server.Serve()
}

func (r *RefImpl) Shutdown() {
	r.server.Stop()
	r.grpcCertManager.Close()
	if r.oauthCertManager != nil { r.oauthCertManager.Close() }
}

func NewAuth(jwksUrl string, tls *tls.Config, scope string) kitNetGrpc.AuthInterceptors {
	interceptor := kitNetGrpc.ValidateJWT(jwksUrl, tls, func(ctx context.Context, method string) kitNetGrpc.Claims {
		return jwt.NewScopeClaims(scope)
	})
	return kitNetGrpc.MakeAuthInterceptors(func(ctx context.Context, method string) (context.Context, error) {
		ctx, err := interceptor(ctx, method)
		if err != nil {
			log.Errorf("auth interceptor: %v", err)
		}
		return ctx, err
	})
}
