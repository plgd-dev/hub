package refImpl

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/security/certificateManager"

	"google.golang.org/grpc"

	"github.com/plgd-dev/cloud/grpc-gateway/service"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security/jwt"
	"google.golang.org/grpc/credentials"
)

type Config struct {
	Log     log.Config
	JwksURL string                    `envconfig:"JWKS_URL"`
	Listen  certificateManager.Config `envconfig:"LISTEN"`
	Dial    certificateManager.Config `envconfig:"DIAL"`
	kitNetGrpc.Config
	service.HandlerConfig
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}

func Init(config Config) (*kitNetGrpc.Server, error) {
	log.Setup(config.Log)
	log.Info(config.String())

	listenCertManager, err := certificateManager.NewCertificateManager(config.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create server cert manager %w", err)
	}
	dialCertManager, err := certificateManager.NewCertificateManager(config.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create client cert manager %w", err)
	}

	auth := NewAuth(config.JwksURL, dialCertManager.GetClientTLSConfig())

	listenTLSConfig := listenCertManager.GetServerTLSConfig()
	server, err := kitNetGrpc.NewServer(config.Addr, grpc.Creds(credentials.NewTLS(listenTLSConfig)), auth.Stream(), auth.Unary())
	if err != nil {
		return nil, err
	}
	server.AddCloseFunc(func() {
		listenCertManager.Close()
		dialCertManager.Close()
	})

	if err := service.AddHandler(server, config.HandlerConfig, dialCertManager.GetClientTLSConfig()); err != nil {
		return nil, err
	}

	return server, nil
}

func NewAuth(jwksUrl string, tls *tls.Config) kitNetGrpc.AuthInterceptors {
	return kitNetGrpc.MakeAuthInterceptors(func(ctx context.Context, method string) (context.Context, error) {
		interceptor := kitNetGrpc.ValidateJWT(jwksUrl, tls, func(ctx context.Context, method string) kitNetGrpc.Claims {
			return jwt.NewScopeClaims()
		})
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
