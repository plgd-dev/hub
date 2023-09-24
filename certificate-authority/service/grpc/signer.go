package grpc

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"github.com/karrick/tparse/v2"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/pkg/security/certificateSigner"
	"github.com/plgd-dev/kit/v2/security"
)

type Signer struct {
	validFrom   func() time.Time
	validFor    time.Duration
	certificate []*x509.Certificate
	privateKey  crypto.PrivateKey
	ownerClaim  string
	hubID       string
}

func isRootCA(cert *x509.Certificate) bool {
	return cert.IsCA && bytes.Equal(cert.RawIssuer, cert.RawSubject) && cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature) == nil
}

func setCAPools(roots *x509.CertPool, intermediates *x509.CertPool, certs []*x509.Certificate) {
	for _, cert := range certs {
		if !cert.IsCA {
			continue
		}
		if isRootCA(cert) {
			roots.AddCert(cert)
			continue
		}
		intermediates.AddCert(cert)
	}
}

func checkCertificatePrivateKey(cert []*x509.Certificate, priv *ecdsa.PrivateKey) error {
	if len(cert) == 0 {
		return fmt.Errorf("at least one certificate need to be set")
	}
	x509Cert := cert[0]
	switch pub := x509Cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		if pub.X.Cmp(priv.X) != 0 || pub.Y.Cmp(priv.Y) != 0 {
			return errors.New("private key does not match public key")
		}
	default:
		return errors.New("unknown public key algorithm")
	}
	return nil
}

func NewSigner(ownerClaim string, hubID string, signerConfig SignerConfig) (*Signer, error) {
	certificate, err := security.LoadX509(signerConfig.CertFile)
	if err != nil {
		return nil, err
	}
	privateKey, err := security.LoadX509PrivateKey(signerConfig.KeyFile)
	if err != nil {
		return nil, err
	}
	if err := checkCertificatePrivateKey(certificate, privateKey); err != nil {
		return nil, err
	}
	if len(certificate) == 1 && isRootCA(certificate[0]) {
		return &Signer{
			validFrom: func() time.Time {
				t, _ := tparse.ParseNow(time.RFC3339, signerConfig.ValidFrom)
				return t
			},
			validFor:    signerConfig.ExpiresIn,
			certificate: certificate,
			privateKey:  privateKey,
			ownerClaim:  ownerClaim,
			hubID:       hubID,
		}, nil
	}

	intermediateCA := x509.NewCertPool()
	rootCA := x509.NewCertPool()
	for _, caFile := range signerConfig.caPoolArray {
		certs, err := security.LoadX509(caFile)
		if err != nil {
			return nil, err
		}
		setCAPools(rootCA, intermediateCA, certs)
	}
	setCAPools(rootCA, intermediateCA, certificate[1:])

	verifyOpts := x509.VerifyOptions{
		Roots:         rootCA,
		Intermediates: intermediateCA,
		CurrentTime:   time.Now(),
	}

	chains, err := certificate[0].Verify(verifyOpts)
	if err != nil {
		return nil, err
	}

	return &Signer{
		validFrom: func() time.Time {
			t, _ := tparse.ParseNow(time.RFC3339, signerConfig.ValidFrom)
			return t
		},
		validFor:    signerConfig.ExpiresIn,
		certificate: chains[0],
		privateKey:  privateKey,
		ownerClaim:  ownerClaim,
		hubID:       hubID,
	}, nil
}

func (s *Signer) Sign(ctx context.Context, csr []byte) ([]byte, *pb.SigningRecord, error) {
	notBefore := s.validFrom()
	notAfter := notBefore.Add(s.validFor)
	var signingRecord *pb.SigningRecord
	signer := certificateSigner.New(s.certificate, s.privateKey, certificateSigner.WithNotBefore(notBefore), certificateSigner.WithNotAfter(notAfter), certificateSigner.WithOverrideCertTemplate(func(template *x509.Certificate) error {
		subject, err := overrideSubject(ctx, template.Subject, s.ownerClaim, s.hubID, "")
		if err != nil {
			return err
		}
		template.Subject = subject
		owner, err := ownerToUUID(ctx, s.ownerClaim)
		if err != nil {
			return err
		}
		signingRecord, err = toSigningRecord(owner, template)
		return err
	}))
	crt, err := signer.Sign(ctx, csr)
	return crt, signingRecord, err
}

func (s *Signer) SignIdentityCSR(ctx context.Context, csr []byte) ([]byte, *pb.SigningRecord, error) {
	notBefore := s.validFrom()
	notAfter := notBefore.Add(s.validFor)
	var signingRecord *pb.SigningRecord
	signer := certificateSigner.NewIdentityCertificateSigner(s.certificate, s.privateKey, certificateSigner.WithNotBefore(notBefore), certificateSigner.WithNotAfter(notAfter), certificateSigner.WithOverrideCertTemplate(func(template *x509.Certificate) error {
		subject, err := overrideSubject(ctx, template.Subject, s.ownerClaim, s.hubID, "uuid:")
		if err != nil {
			return err
		}
		template.Subject = subject
		owner, err := ownerToUUID(ctx, s.ownerClaim)
		if err != nil {
			return err
		}
		signingRecord, err = toSigningRecord(owner, template)
		return err
	}))
	cert, err := signer.Sign(ctx, csr)
	return cert, signingRecord, err
}
