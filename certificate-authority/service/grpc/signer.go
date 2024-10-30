package grpc

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/karrick/tparse/v2"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/certificate-authority/service/uri"
	"github.com/plgd-dev/hub/v2/pkg/security/certificateSigner"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
)

type Signer struct {
	validFrom   func() time.Time
	validFor    time.Duration
	certificate []*x509.Certificate
	privateKey  *ecdsa.PrivateKey
	issuerID    string
	ownerClaim  string
	hubID       string
	crl         struct {
		serverAddress string
		validFor      time.Duration
	}
}

func checkCertificatePrivateKey(cert []*x509.Certificate, priv *ecdsa.PrivateKey) error {
	if len(cert) == 0 {
		return errors.New("at least one certificate need to be set")
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

func getIssuerID(rootCertificate *x509.Certificate) (string, error) {
	publicKeyRaw, err := x509.MarshalPKIXPublicKey(rootCertificate.PublicKey)
	if err != nil {
		return "", err
	}
	return uuid.NewSHA1(uuid.NameSpaceX500, publicKeyRaw).String(), nil
}

func newSigner(ownerClaim, hubID, crlServerAddress string, signerConfig SignerConfig, privateKey *ecdsa.PrivateKey, certificate []*x509.Certificate) (*Signer, error) {
	issuerID, err := getIssuerID(certificate[0])
	if err != nil {
		return nil, err
	}
	signer := &Signer{
		validFrom: func() time.Time {
			t, _ := tparse.ParseNow(time.RFC3339, signerConfig.ValidFrom)
			return t
		},
		validFor:    signerConfig.ExpiresIn,
		certificate: certificate,
		privateKey:  privateKey,
		issuerID:    issuerID,
		ownerClaim:  ownerClaim,
		hubID:       hubID,
	}
	if signerConfig.CRL.Enabled {
		if err = pkgX509.ValidateCRLDistributionPointAddress(crlServerAddress); err != nil {
			return nil, err
		}
		signer.crl.serverAddress = crlServerAddress
		signer.crl.validFor = signerConfig.CRL.ExpiresIn
	}
	return signer, nil
}

func NewSigner(ownerClaim, hubID, crlServerAddress string, signerConfig SignerConfig) (*Signer, error) {
	data, err := signerConfig.CertFile.Read()
	if err != nil {
		return nil, err
	}
	certificate, err := pkgX509.ParseX509(data)
	if err != nil {
		return nil, err
	}
	data, err = signerConfig.KeyFile.Read()
	if err != nil {
		return nil, err
	}
	privateKey, err := pkgX509.ParsePrivateKey(data)
	if err != nil {
		return nil, err
	}
	if err = checkCertificatePrivateKey(certificate, privateKey); err != nil {
		return nil, err
	}
	if len(certificate) == 1 && pkgX509.IsRootCA(certificate[0]) {
		return newSigner(ownerClaim, hubID, crlServerAddress, signerConfig, privateKey, certificate)
	}
	certificateAuthorities := make([]*x509.Certificate, 0, len(signerConfig.caPoolArray)*4)
	for _, caFile := range signerConfig.caPoolArray {
		data, errR := caFile.Read()
		if errR != nil {
			return nil, errR
		}
		certs, errR := pkgX509.ParseX509(data)
		if errR != nil {
			return nil, errR
		}
		certificateAuthorities = append(certificateAuthorities, certs...)
	}
	certificateAuthorities = append(certificateAuthorities, certificate[1:]...)
	verifyOpts := x509.VerifyOptions{
		CurrentTime: time.Now(),
	}
	chains, err := pkgX509.Verify(certificate, certificateAuthorities, false, verifyOpts)
	if err != nil {
		return nil, err
	}
	return newSigner(ownerClaim, hubID, crlServerAddress, signerConfig, privateKey, chains[0])
}

func (s *Signer) prepareSigningRecord(ctx context.Context, template *x509.Certificate) (*pb.SigningRecord, error) {
	subject, err := overrideSubject(ctx, template.Subject, s.ownerClaim, s.hubID, "")
	if err != nil {
		return nil, err
	}
	template.Subject = subject
	owner, err := ownerToUUID(ctx, s.ownerClaim)
	if err != nil {
		return nil, err
	}
	return toSigningRecord(owner, s.issuerID, template)
}

func (s *Signer) GetCertificate() *x509.Certificate {
	return s.certificate[0]
}

func (s *Signer) GetPrivateKey() *ecdsa.PrivateKey {
	return s.privateKey
}

func (s *Signer) GetIssuerID() string {
	return s.issuerID
}

func (s *Signer) GetCRLConfiguration() (string, time.Duration) {
	return s.crl.serverAddress, s.crl.validFor
}

func (s *Signer) IsCRLEnabled() bool {
	return s.crl.serverAddress != ""
}

func (s *Signer) newCertificateSigner(identitySigner bool, opts ...func(cfg *certificateSigner.SignerConfig)) (*certificateSigner.CertificateSigner, error) {
	if identitySigner {
		return certificateSigner.NewIdentityCertificateSigner(s.certificate, s.privateKey, opts...)
	}
	return certificateSigner.New(s.certificate, s.privateKey, opts...)
}

func (s *Signer) sign(ctx context.Context, isIdentityCertificate bool, csr []byte) ([]byte, *pb.SigningRecord, error) {
	notBefore := s.validFrom()
	notAfter := notBefore.Add(s.validFor)
	var signingRecord *pb.SigningRecord
	opts := []certificateSigner.Opt{
		certificateSigner.WithNotBefore(notBefore),
		certificateSigner.WithNotAfter(notAfter),
		certificateSigner.WithOverrideCertTemplate(func(template *x509.Certificate) error {
			var err error
			signingRecord, err = s.prepareSigningRecord(ctx, template)
			return err
		}),
	}
	if s.IsCRLEnabled() {
		dp := []string{s.crl.serverAddress, uri.SigningRevocationListBase, s.issuerID}
		opts = append(opts, certificateSigner.WithCRLDistributionPoints([]string{strings.Join(dp, "/")}))
	}
	signer, err := s.newCertificateSigner(isIdentityCertificate, opts...)
	if err != nil {
		return nil, nil, err
	}
	cert, err := signer.Sign(ctx, csr)
	if err != nil {
		return nil, nil, err
	}
	return cert, signingRecord, nil
}

func (s *Signer) Sign(ctx context.Context, csr []byte) ([]byte, *pb.SigningRecord, error) {
	return s.sign(ctx, false, csr)
}

func (s *Signer) SignIdentityCSR(ctx context.Context, csr []byte) ([]byte, *pb.SigningRecord, error) {
	return s.sign(ctx, true, csr)
}
