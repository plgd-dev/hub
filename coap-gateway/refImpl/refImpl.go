package refImpl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-ocf/kit/security/certManager"

	"github.com/go-ocf/cloud/coap-gateway/service"
	"github.com/go-ocf/cloud/coap-gateway/uri"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/kit/log"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
)

type Config struct {
	Service          service.Config
	Dial             certManager.Config    `envconfig:"DIAL"`
	Listen           certManager.OcfConfig `envconfig:"LISTEN"`
	ListenWithoutTLS bool                  `envconfig:"LISTEN_WITHOUT_TLS"`
	Log              log.Config            `envconfig:"LOG"`
}

type RefImpl struct {
	service           *service.Server
	dialCertManager   certManager.CertManager
	listenCertManager certManager.CertManager
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

// Init creates reference implementation for coap-gateway with default authorization interceptor.
func Init(config Config) (*RefImpl, error) {
	dialCertManager, err := certManager.NewCertManager(config.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create dial cert manager %v", err)
	}
	auth := NewDefaultAuthInterceptor()
	return InitWithAuthInterceptor(config, dialCertManager, auth)
}

// InitWithAuthInterceptor creates reference implementation for coap-gateway with custom authorization interceptor.
func InitWithAuthInterceptor(config Config, dialCertManager certManager.CertManager, auth kitNetCoap.Interceptor) (*RefImpl, error) {
	log.Setup(config.Log)

	log.Info(config.String())

	var listenCertManager certManager.CertManager
	var err error
	if !config.ListenWithoutTLS {
		listenCertManager, err = certManager.NewOcfCertManager(config.Listen)
		if err != nil {
			dialCertManager.Close()
			return nil, fmt.Errorf("cannot create listen cert manager %v", err)
		}
	}

	return &RefImpl{
		service:           service.New(config.Service, dialCertManager, listenCertManager, auth),
		dialCertManager:   dialCertManager,
		listenCertManager: listenCertManager,
	}, nil
}

// Serve starts handling coap requests.
func (r *RefImpl) Serve() error {
	return r.service.Serve()
}

// Shutdown shutdowns the service.
func (r *RefImpl) Shutdown() error {
	err := r.service.Shutdown()
	r.dialCertManager.Close()
	if r.listenCertManager != nil {
		r.listenCertManager.Close()
	}
	return err
}

// NewDefaultAuthInterceptor creates default authorization interceptor.
func NewDefaultAuthInterceptor() kitNetCoap.Interceptor {
	return func(ctx context.Context, code coapCodes.Code, path string) (context.Context, error) {
		switch path {
		case uri.RefreshToken, uri.SecureRefreshToken, uri.SignUp, uri.SecureSignUp, uri.SignIn, uri.SecureSignIn, uri.ResourcePing:
			return ctx, nil
		}
		_, err := kitNetCoap.TokenFromCtx(ctx)
		if err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}
