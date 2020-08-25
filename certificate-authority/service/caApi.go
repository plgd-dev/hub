package service

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/go-ocf/cloud/certificate-authority/pb"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/security"
	"github.com/karrick/tparse/v2"
	"google.golang.org/grpc"
)

type CertificateSigner interface {
	//csr is encoded by PEM and returns PEM
	Sign(ctx context.Context, csr []byte) ([]byte, error)
}

// RequestHandler handles incoming requests.
type RequestHandler struct {
	ValidFrom   func() time.Time
	ValidFor    time.Duration
	Certificate []*x509.Certificate
	PrivateKey  crypto.PrivateKey
}

type ValidFromDecoder func() time.Time

func (d *ValidFromDecoder) Decode(value string) error {
	if value == "" {
		*d = func() time.Time {
			return time.Now().Add(time.Hour * -1)
		}
		return nil
	}
	_, err := tparse.ParseNow(time.RFC3339, value)
	if err != nil {
		return fmt.Errorf("invalid VALID_FROM(%v): %v", value, err)
	}
	*d = func() time.Time {
		t, _ := tparse.ParseNow(time.RFC3339, value)
		return t
	}
	return nil
}

type SignerConfig struct {
	Certificate   string           `envconfig:"CERTIFICATE"`
	PrivateKey    string           `envconfig:"PRIVATE_KEY"`
	ValidFrom     ValidFromDecoder `envconfig:"VALID_FROM" default:"now"`
	ValidDuration time.Duration    `envconfig:"VALID_DURATION" default:"87600h"`
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

	return NewRequestHandler(cfg.ValidFrom, cfg.ValidDuration, chainCerts, privateKey), nil
}

// NewRequestHandler factory for new RequestHandler.
func NewRequestHandler(
	ValidFrom func() time.Time,
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
