package refImpl

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/plgd-dev/kit/security/certificateManager"

	"google.golang.org/grpc"

	"github.com/plgd-dev/cloud/resource-directory/service"
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
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

// StreamServerInterceptor returns a new unary server interceptors that performs per-request auth.
func StreamServerInterceptor(authFunc func(ctx context.Context, method string) (context.Context, error)) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		var newCtx context.Context
		var err error
		newCtx, err = authFunc(stream.Context(), info.FullMethod)
		if err != nil {
			return err
		}
		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = newCtx
		return handler(srv, wrapped)
	}
}

// UnaryServerInterceptor returns a new unary server interceptors that performs per-request auth.
func UnaryServerInterceptor(authFunc func(ctx context.Context, method string) (context.Context, error)) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var newCtx context.Context
		var err error
		newCtx, err = authFunc(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}
		return handler(newCtx, req)
	}
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
	server, err := kitNetGrpc.NewServer(config.Addr, grpc.Creds(credentials.NewTLS(listenTLSConfig)), /*
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
				grpc_zap.StreamServerInterceptor(logger),
				StreamServerInterceptor(authFunc),
			)),
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
				grpc_zap.UnaryServerInterceptor(logger),
				UnaryServerInterceptor(authFunc),
			)),*/
		auth.Stream(), auth.Unary(),
	)

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

func makeAuthFunc(jwksUrl string, tls *tls.Config) func(ctx context.Context, method string) (context.Context, error) {
	return func(ctx context.Context, method string) (context.Context, error) {
		switch method {
		case "/ocf.cloud.grpcgateway.pb.GrpcGateway/GetClientConfiguration":
			return ctx, nil
		}
		interceptor := kitNetGrpc.ValidateJWT(jwksUrl, tls, func(ctx context.Context, method string) kitNetGrpc.Claims {
			return jwt.NewScopeClaims()
		})
		token, _ := kitNetGrpc.TokenFromMD(ctx)
		ctx, err := interceptor(ctx, method)
		if err != nil {
			log.Errorf("auth interceptor %v %v: %v", method, token, err)
			return ctx, err
		}
		userID, err := kitNetGrpc.UserIDFromMD(ctx)
		if err != nil {
			userID, err = kitNetGrpc.UserIDFromTokenMD(ctx)
			if err == nil {
				ctx = kitNetGrpc.CtxWithIncomingUserID(ctx, userID)
			}
		}
		if err != nil {
			log.Errorf("auth cannot get userID: %v", err)
			return ctx, err
		}
		return kitNetGrpc.CtxWithUserID(ctx, userID), nil
	}
}

func NewAuth(jwksUrl string, tls *tls.Config) kitNetGrpc.AuthInterceptors {
	return kitNetGrpc.MakeAuthInterceptors(makeAuthFunc(jwksUrl, tls))
}
