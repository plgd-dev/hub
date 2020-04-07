package refImpl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-ocf/kit/security/certManager"

	"github.com/go-ocf/ocf-cloud/certificate-authority/pb"
	"github.com/go-ocf/ocf-cloud/certificate-authority/service"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Config struct {
	Log     log.Config
	Service service.Config
	Signer  service.SignerConfig
	Listen  certManager.Config `envconfig:"LISTEN"`
	JwksURL string             `envconfig:"JWKS_URL" required:"True"`
}

type RefImpl struct {
	handle            *service.RequestHandler
	server            *kitNetGrpc.Server
	listenCertManager certManager.CertManager
}

// NewRequestHandlerFromConfig creates RegisterGrpcGatewayServer with all dependencies.
func NewRefImplFromConfig(config Config, auth kitNetGrpc.AuthInterceptors) (*RefImpl, error) {
	listenCertManager, err := certManager.NewCertManager(config.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create listen cert manager %v", err)
	}

	serverTLSConfig := listenCertManager.GetServerTLSConfig()
	svr, err := kitNetGrpc.NewServer(config.Service.Addr, grpc.Creds(credentials.NewTLS(&serverTLSConfig)), auth.Stream(), auth.Unary())
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
	auth := kitNetGrpc.MakeAuthInterceptors(func(ctx context.Context, method string) (context.Context, error) {
		return ctx, nil
	})
	return InitWithAuth(config, auth)
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
}
