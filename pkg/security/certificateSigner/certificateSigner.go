package certificateSigner

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
	"slices"
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/kit/v2/security"
)

type SignerConfig struct {
	ValidNotBefore        time.Time
	ValidNotAfter         time.Time
	CRLDistributionPoints []string
	OverrideCertTemplate  func(template *x509.Certificate) error
}

type Opt = func(cfg *SignerConfig)

func WithNotBefore(validNotBefore time.Time) Opt {
	return func(cfg *SignerConfig) {
		cfg.ValidNotBefore = validNotBefore
	}
}

func WithNotAfter(validNotAfter time.Time) Opt {
	return func(cfg *SignerConfig) {
		cfg.ValidNotAfter = validNotAfter
	}
}

func WithCRLDistributionPoints(crlDistributionPoints []string) Opt {
	return func(cfg *SignerConfig) {
		cfg.CRLDistributionPoints = slices.Clone(crlDistributionPoints)
	}
}

func WithOverrideCertTemplate(overrideCertTemplate func(template *x509.Certificate) error) Opt {
	return func(cfg *SignerConfig) {
		cfg.OverrideCertTemplate = overrideCertTemplate
	}
}

type CertificateSigner struct {
	caCert []*x509.Certificate
	caKey  crypto.PrivateKey
	cfg    SignerConfig
}

func New(caCert []*x509.Certificate, caKey crypto.PrivateKey, opts ...Opt) *CertificateSigner {
	cfg := SignerConfig{
		ValidNotAfter: pkgTime.MaxTime,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &CertificateSigner{caCert: caCert, caKey: caKey, cfg: cfg}
}

func parseCertificateRequest(csr []byte) (*x509.CertificateRequest, error) {
	csrBlock, _ := pem.Decode(csr)
	if csrBlock == nil {
		return nil, errors.New("pem not found")
	}

	certificateRequest, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		return nil, err
	}

	err = certificateRequest.CheckSignature()
	if err != nil {
		return nil, err
	}
	return certificateRequest, nil
}

func (s *CertificateSigner) Sign(_ context.Context, csr []byte) ([]byte, error) {
	if len(s.caCert) == 0 {
		return nil, errors.New("cannot sign with empty signer CA certificates")
	}
	parsedCSR, err := parseCertificateRequest(csr)
	if err != nil {
		return nil, err
	}
	notBefore := s.cfg.ValidNotBefore
	notAfter := s.cfg.ValidNotAfter
	for _, c := range s.caCert {
		if notBefore.Before(c.NotBefore) {
			notBefore = c.NotBefore
		}
		if notAfter.After(c.NotAfter) {
			notAfter = c.NotAfter
		}
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		Subject:               parsedCSR.Subject,
		PublicKeyAlgorithm:    parsedCSR.PublicKeyAlgorithm,
		PublicKey:             parsedCSR.PublicKey,
		SignatureAlgorithm:    s.caCert[0].SignatureAlgorithm,
		DNSNames:              parsedCSR.DNSNames,
		IPAddresses:           parsedCSR.IPAddresses,
		URIs:                  parsedCSR.URIs,
		EmailAddresses:        parsedCSR.EmailAddresses,
		ExtraExtensions:       parsedCSR.Extensions,
		CRLDistributionPoints: s.cfg.CRLDistributionPoints,
	}
	if s.cfg.OverrideCertTemplate != nil {
		if err = s.cfg.OverrideCertTemplate(&template); err != nil {
			return nil, err
		}
	}
	signedCsr, err := x509.CreateCertificate(rand.Reader, &template, s.caCert[0], parsedCSR.PublicKey, s.caKey)
	if err != nil {
		return nil, err
	}
	return security.CreatePemChain(s.caCert, signedCsr)
}
