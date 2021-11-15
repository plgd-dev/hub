package service

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/karrick/tparse/v2"
	"github.com/plgd-dev/hub/certificate-authority/pb"
	"github.com/plgd-dev/hub/pkg/net/grpc/server"
	"github.com/plgd-dev/kit/v2/security"
	"google.golang.org/grpc"
)

type CertificateSigner interface {
	//csr is encoded by PEM and returns PEM
	Sign(ctx context.Context, csr []byte) ([]byte, error)
}

// RequestHandler handles incoming requests.
type RequestHandler struct {
	pb.UnimplementedCertificateAuthorityServer
	ValidFrom   func() time.Time
	ValidFor    time.Duration
	Certificate []*x509.Certificate
	PrivateKey  crypto.PrivateKey
	Config      Config
}

func AddHandler(svr *server.Server, cfg Config) error {
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

func NewRequestHandlerFromConfig(cfg Config) (*RequestHandler, error) {
	chainCerts, err := security.LoadX509(cfg.Signer.CertFile)
	if err != nil {
		return nil, err
	}
	privateKey, err := security.LoadX509PrivateKey(cfg.Signer.KeyFile)
	if err != nil {
		return nil, err
	}

	return NewRequestHandler(func() time.Time {
		t, _ := tparse.ParseNow(time.RFC3339, cfg.Signer.ValidFrom)
		return t
	}, cfg.Signer.ExpiresIn, chainCerts, privateKey, cfg), nil
}

// NewRequestHandler factory for new RequestHandler.
func NewRequestHandler(
	ValidFrom func() time.Time,
	ValidFor time.Duration,
	Certificate []*x509.Certificate,
	PrivateKey crypto.PrivateKey,
	cfg Config,
) *RequestHandler {
	return &RequestHandler{
		ValidFrom:   ValidFrom,
		ValidFor:    ValidFor,
		Certificate: Certificate,
		PrivateKey:  PrivateKey,
		Config:      cfg,
	}
}
