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
	"github.com/plgd-dev/kit/security/jwt"

	"github.com/plgd-dev/cloud/certificate-authority/pb"
	"github.com/plgd-dev/cloud/certificate-authority/service"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Config struct {
	Log     log.Config           `envconfig:"LOG"`
	Signer  service.SignerConfig `envconfig:"SIGNER"`
	Listen  certManager.Config   `envconfig:"LISTEN"`
	Dial    certManager.Config   `envconfig:"DIAL"`
	JwksURL string               `envconfig:"JWKS_URL"`
	kitNetGrpc.Config
}

type RefImpl struct {
	handle            *service.RequestHandler
	server            *kitNetGrpc.Server
	listenCertManager certManager.CertManager
	dialCertManager   certManager.CertManager
}

// NewRequestHandlerFromConfig creates RegisterGrpcGatewayServer with all dependencies.
func NewRefImplFromConfig(config Config, auth kitNetGrpc.AuthInterceptors) (*RefImpl, error) {
	logger, err := log.NewLogger(config.Log)
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)
	log.Info(config.String())
	listenCertManager, err := certManager.NewCertManager(config.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create listen cert manager %w", err)
	}

	listenTLSConfig := listenCertManager.GetServerTLSConfig()

	svr, err := kitNetGrpc.NewServer(config.Addr, grpc.Creds(credentials.NewTLS(listenTLSConfig)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.StreamServerInterceptor(logger),
			auth.Stream(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(logger),
			auth.Unary(),
		)),
	)
	if err != nil {
		listenCertManager.Close()
		return nil, err
	}

	handler, err := service.NewRequestHandlerFromConfig(config.Signer)
	if err != nil {
		return nil, err
	}
	return &RefImpl{
		handle:            handler,
		listenCertManager: listenCertManager,
		server:            svr,
	}, nil
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

func Init(config Config) (*RefImpl, error) {
	dialCertManager, err := certManager.NewCertManager(config.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create dial cert manager %w", err)
	}

	auth := NewAuth(config.JwksURL, dialCertManager.GetClientTLSConfig(), "openid")
	r, err := InitWithAuth(config, auth)
	if err != nil {
		dialCertManager.Close()
		return nil, err
	}
	r.dialCertManager = dialCertManager
	return r, nil
}

func InitWithAuth(config Config, auth kitNetGrpc.AuthInterceptors) (*RefImpl, error) {
	log.Setup(config.Log)
	log.Info(config.String())

	impl, err := NewRefImplFromConfig(config, auth)
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
	r.listenCertManager.Close()
	if r.dialCertManager != nil {
		r.dialCertManager.Close()
	}
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
