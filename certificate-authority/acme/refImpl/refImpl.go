package refImpl

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-ocf/cloud/certificate-authority/acme/service"
	kitNet "github.com/go-ocf/kit/net"
	"github.com/go-ocf/kit/security/generateCertificate"

	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/security"
	ocfSigner "github.com/go-ocf/kit/security/signer"
)

type Config struct {
	Log                 log.Config
	Addr                string        `envconfig:"ADDR" default:"0.0.0.0:10443"`
	FQDN                string        `envconfig:"FQDN" default:"acme.ca.ocf.cloud"`
	Domains             []string      `envconfig:"DOMAINS"`
	AcmeDBDir           string        `envconfig:"ACME_DB_DIR"`
	SignerCertificate   string        `envconfig:"SIGNER_CERTIFICATE"`
	SignerPrivateKey    string        `envconfig:"SIGNER_PRIVATE_KEY"`
	SignerValidDuration time.Duration `envconfig:"SIGNER_VALID_DURATION" default:"87600h"`
}

type RefImpl struct {
	listenTLS  tls.Config
	listener   net.Listener
	server     *http.Server
	selfSigner *selfSigner
}

// NewRefImplFromConfig creates RegisterGrpcGatewayServer with all dependencies.
func NewRefImplFromConfig(config Config) (*RefImpl, error) {
	var addr kitNet.Addr
	addr, err := kitNet.ParseString("", config.Addr)
	if err != nil {
		return nil, err
	}

	chainCerts, err := security.LoadX509(config.SignerCertificate)
	if err != nil {
		return nil, err
	}
	privateKey, err := security.LoadX509PrivateKey(config.SignerPrivateKey)
	if err != nil {
		return nil, err
	}

	listenCert, err := getSelfCertificate(config.FQDN, config.Domains, chainCerts, privateKey)
	if err != nil {
		return nil, err
	}

	// Create the main listener.
	l, err := tls.Listen("tcp", config.Addr, &tls.Config{
		Certificates: []tls.Certificate{listenCert},
		ClientAuth:   tls.NoClientCert,
	})
	if err != nil {
		return nil, err
	}

	signer := ocfSigner.NewBasicCertificateSigner(chainCerts, privateKey, config.SignerValidDuration)

	selfSigner := &selfSigner{
		pemSigner: signer,
	}

	h, err := service.NewHandler(config.FQDN, config.AcmeDBDir, addr.GetPort(), []service.Signer{selfSigner})
	if err != nil {
		return nil, err
	}

	return &RefImpl{
		server: &http.Server{
			Handler: h,
		},
		listener: l,
	}, nil
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

func Init(config Config) (*RefImpl, error) {
	log.Setup(config.Log)
	log.Info(config.String())

	impl, err := NewRefImplFromConfig(config)
	if err != nil {
		return nil, err
	}

	return impl, nil
}

func getSelfCertificate(FQDN string, domains []string, chainCerts []*x509.Certificate, privateKey *ecdsa.PrivateKey) (tls.Certificate, error) {
	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = FQDN
	cfg.SubjectAlternativeName.DNSNames = domains
	cfg.ExtensionKeyUsages = []string{"server", "client"}
	cfg.ValidFor = time.Hour * 86400
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	pemCert, err := generateCertificate.GenerateCert(cfg, priv, chainCerts, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	b, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	return tls.X509KeyPair(pemCert, pemKey)
}

func (r *RefImpl) Serve() error {
	return r.server.Serve(r.listener)
}

func (r *RefImpl) Shutdown() {
	r.listener.Close()
	r.server.Shutdown(context.Background())
}

type signer = interface {
	Sign(ctx context.Context, csr []byte) ([]byte, error)
}

type selfSigner struct {
	pemSigner signer
}

func (s *selfSigner) ID() string {
	return "proxy"
}

func (s *selfSigner) Sign(ctx context.Context, csr []byte) ([]byte, error) {
	return s.pemSigner.Sign(ctx, csr)
}
