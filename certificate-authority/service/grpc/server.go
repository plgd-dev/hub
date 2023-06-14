package grpc

import (
	"context"
	"crypto"
	"crypto/x509"
	"time"

	"github.com/karrick/tparse/v2"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/certificate-authority/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/kit/v2/security"
)

type CertificateSigner interface {
	// csr is encoded by PEM and returns PEM
	Sign(ctx context.Context, csr []byte) ([]byte, error)
}

// CertificateAuthorityServer handles incoming requests.
type CertificateAuthorityServer struct {
	pb.UnimplementedCertificateAuthorityServer

	validFrom    func() time.Time
	validFor     time.Duration
	certificate  []*x509.Certificate
	privateKey   crypto.PrivateKey
	signerConfig SignerConfig
	logger       log.Logger
	ownerClaim   string
	store        *mongodb.Store
}

func NewCertificateAuthorityServer(ownerClaim string, signerConfig SignerConfig, store *mongodb.Store, logger log.Logger) (*CertificateAuthorityServer, error) {
	certificate, err := security.LoadX509(signerConfig.CertFile)
	if err != nil {
		return nil, err
	}
	privateKey, err := security.LoadX509PrivateKey(signerConfig.KeyFile)
	if err != nil {
		return nil, err
	}

	return &CertificateAuthorityServer{
		validFrom: func() time.Time {
			t, _ := tparse.ParseNow(time.RFC3339, signerConfig.ValidFrom)
			return t
		},
		validFor:     signerConfig.ExpiresIn,
		certificate:  certificate,
		privateKey:   privateKey,
		signerConfig: signerConfig,
		logger:       logger,
		ownerClaim:   ownerClaim,
		store:        store,
	}, nil
}
