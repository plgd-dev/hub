package refImpl

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"

	"github.com/plgd-dev/kit/security/certManager"

	"google.golang.org/grpc"

	"github.com/plgd-dev/cloud/grpc-gateway/service"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/pkg/security/jwt"
	"github.com/plgd-dev/kit/log"
	"google.golang.org/grpc/credentials"
)

type Config struct {
	Log     log.Config
	JwksURL string             `envconfig:"JWKS_URL"`
	Listen  certManager.Config `envconfig:"LISTEN"`
	Dial    certManager.Config `envconfig:"DIAL"`
	Addr    string             `envconfig:"ADDRESS" env:"ADDRESS" long:"address" default:"0.0.0.0:9100"`
	service.Config
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

func Init(config Config) (*server.Server, error) {
	logger, err := log.NewLogger(config.Log)
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)
	log.Info(config.String())

	listenCertManager, err := certManager.NewCertManager(config.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create server cert manager %w", err)
	}
	dialCertManager, err := certManager.NewCertManager(config.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create client cert manager %w", err)
	}

	auth := NewAuth(config.JwksURL, dialCertManager.GetClientTLSConfig())
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

	listenTLSConfig := listenCertManager.GetServerTLSConfig()
	server, err := server.NewServer(config.Addr, grpc.Creds(credentials.NewTLS(listenTLSConfig)),
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
		listenCertManager.Close()
		dialCertManager.Close()
	})

	if err := service.AddHandler(server, config.Config, dialCertManager.GetClientTLSConfig()); err != nil {
		return nil, err
	}

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
