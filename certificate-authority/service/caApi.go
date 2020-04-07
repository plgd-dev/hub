package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ocf/ocf-cloud/certificate-authority/pb"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/security"
	"github.com/go-ocf/kit/security/signer"
	"google.golang.org/grpc"
)

type CertificateSigner interface {
	//csr is encoded by PEM and returns PEM
	Sign(ctx context.Context, csr []byte) ([]byte, error)
}

// RequestHandler handles incoming requests.
type RequestHandler struct {
	identitySigner CertificateSigner
	signer         CertificateSigner
}

type SignerConfig struct {
	Certificate   string        `envconfig:"SIGNER_CERTIFICATE" required:"True"`
	PrivateKey    string        `envconfig:"SIGNER_PRIVATE_KEY" required:"True"`
	ValidDuration time.Duration `envconfig:"SIGNER_VALID_DURATION" default:"87600h"`
}

func AddHandler(svr *kitNetGrpc.Server, cfg SignerConfig) error {
	handler, err := NewRequestHandlerFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("could not create go-ocf/certificate-authority: %v", err)
	}
	pb.RegisterCertificateAuthorityServer(svr.Server, handler)
	return nil
}

// Register registers the handler instance with a gRPC server.
func Register(server *grpc.Server, handler *RequestHandler) {
	pb.RegisterCertificateAuthorityServer(server, handler)
}

func NewRequestHandlerFromConfig(cfg SignerConfig) (*RequestHandler, error) {
	chainCerts, err := security.LoadX509(cfg.Certificate)
	if err != nil {
		return nil, err
	}
	privateKey, err := security.LoadX509PrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}

	identity := signer.NewIdentityCertificateSigner(chainCerts, privateKey, cfg.ValidDuration)
	basic := signer.NewBasicCertificateSigner(chainCerts, privateKey, cfg.ValidDuration)
	return NewRequestHandler(basic, identity), nil
}

// NewRequestHandler factory for new RequestHandler.
func NewRequestHandler(signer, identitySigner CertificateSigner) *RequestHandler {
	return &RequestHandler{
		signer:         signer,
		identitySigner: identitySigner,
	}
}

func logAndReturnError(err error) error {
	log.Errorf("%v", err)
	return err
}
