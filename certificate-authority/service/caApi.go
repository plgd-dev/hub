package service

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/karrick/tparse/v2"
	"github.com/plgd-dev/cloud/certificate-authority/pb"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security"
	"google.golang.org/grpc"
)

type CertificateSigner interface {
	//csr is encoded by PEM and returns PEM
	Sign(ctx context.Context, csr []byte) ([]byte, error)
}

// RequestHandler handles incoming requests.
type RequestHandler struct {
	ValidFrom   string
	ValidFor    time.Duration
	Certificate []*x509.Certificate
	PrivateKey  crypto.PrivateKey
}

type SignerConfig struct {
	Certificate   string        `envconfig:"CERTIFICATE"`
	PrivateKey    string        `envconfig:"PRIVATE_KEY"`
	ValidFrom     string        `envconfig:"VALID_FROM" default:"now-1h"`
	ValidDuration time.Duration `envconfig:"VALID_DURATION" default:"87600h"`
}

func AddHandler(svr *kitNetGrpc.Server, cfg SignerConfig) error {
	handler, err := NewRequestHandlerFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("could not create plgd-dev/certificate-authority: %w", err)
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
	_, err = tparse.ParseNow(time.RFC3339, cfg.ValidFrom)
	if err != nil {
		return nil, fmt.Errorf("invalid VALID_FROM(%v): %v", cfg.ValidFrom, err)
	}

	return NewRequestHandler(cfg.ValidFrom, cfg.ValidDuration, chainCerts, privateKey), nil
}

// NewRequestHandler factory for new RequestHandler.
func NewRequestHandler(
	ValidFrom string,
	ValidFor time.Duration,
	Certificate []*x509.Certificate,
	PrivateKey crypto.PrivateKey) *RequestHandler {
	return &RequestHandler{
		ValidFrom:   ValidFrom,
		ValidFor:    ValidFor,
		Certificate: Certificate,
		PrivateKey:  PrivateKey,
	}
}

func logAndReturnError(err error) error {
	log.Errorf("%v", err)
	return err
}
