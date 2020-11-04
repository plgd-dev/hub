package refImpl

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/security/certificateManager"
	"github.com/plgd-dev/kit/security/jwt"

	"github.com/plgd-dev/cloud/certificate-authority/pb"
	"github.com/plgd-dev/cloud/certificate-authority/service"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Config struct {
	Log     log.Config                `envconfig:"LOG"`
	Signer  service.SignerConfig      `envconfig:"SIGNER"`
	Listen  certificateManager.Config `envconfig:"LISTEN"`
	Dial    certificateManager.Config `envconfig:"DIAL"`
	JwksURL string                    `envconfig:"JWKS_URL"`
	kitNetGrpc.Config
}

type RefImpl struct {
	handle            *service.RequestHandler
	server            *kitNetGrpc.Server
	listenCertManager *certificateManager.CertificateManager
	dialCertManager   *certificateManager.CertificateManager
}

// NewRequestHandlerFromConfig creates RegisterGrpcGatewayServer with all dependencies.
func NewRefImplFromConfig(config Config, auth kitNetGrpc.AuthInterceptors) (*RefImpl, error) {
	listenCertManager, err := certificateManager.NewCertificateManager(config.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create listen cert manager %w", err)
	}

	listenTLSConfig := listenCertManager.GetServerTLSConfig()
	svr, err := kitNetGrpc.NewServer(config.Addr, grpc.Creds(credentials.NewTLS(listenTLSConfig)), auth.Stream(), auth.Unary())
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
	return config.ToString(c)
}

func Init(config Config) (*RefImpl, error) {
	dialCertManager, err := certificateManager.NewCertificateManager(config.Dial)
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
	return kitNetGrpc.MakeAuthInterceptors(func(ctx context.Context, method string) (context.Context, error) {
		interceptor := kitNetGrpc.ValidateJWT(jwksUrl, tls, func(ctx context.Context, method string) kitNetGrpc.Claims {
			return jwt.NewScopeClaims(scope)
		})
		ctx, err := interceptor(ctx, method)
		if err != nil {
			log.Errorf("auth interceptor: %v", err)
		}
		return ctx, err
	})
}
