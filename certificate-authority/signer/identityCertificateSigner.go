package signer

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

type IdentityCertificateSigner struct {
	caCert         []*x509.Certificate
	caKey          crypto.PrivateKey
	validNotBefore time.Time
	validNotAfter  time.Time
}

func NewIdentityCertificateSigner(caCert []*x509.Certificate, caKey crypto.PrivateKey, validNotBefore time.Time, validNotAfter time.Time) *IdentityCertificateSigner {
	return &IdentityCertificateSigner{caCert: caCert, caKey: caKey, validNotBefore: validNotBefore, validNotAfter: validNotAfter}
}

func (s *IdentityCertificateSigner) Sign(ctx context.Context, csr []byte) (signedCsr []byte, err error) {
	csrBlock, _ := pem.Decode(csr)
	if csrBlock == nil {
		err = fmt.Errorf("pem not found")
		return
	}

	certificateRequest, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		return
	}

	err = certificateRequest.CheckSignature()
	if err != nil {
		return
	}

	notBefore := s.validNotBefore
	notAfter := s.validNotAfter
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	template := x509.Certificate{
		SerialNumber:       serialNumber,
		NotBefore:          notBefore,
		NotAfter:           notAfter,
		Subject:            certificateRequest.Subject,
		PublicKeyAlgorithm: certificateRequest.PublicKeyAlgorithm,
		PublicKey:          certificateRequest.PublicKey,
		SignatureAlgorithm: s.caCert[0].SignatureAlgorithm,
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		UnknownExtKeyUsage: []asn1.ObjectIdentifier{asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 44924, 1, 6}},
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	if len(s.caCert) == 0 {
		return nil, fmt.Errorf("cannot sign with empty signer CA certificates")
	}
	signedCsr, err = x509.CreateCertificate(rand.Reader, &template, s.caCert[0], certificateRequest.PublicKey, s.caKey)
	if err != nil {
		return
	}
	return createPemChain(s.caCert, signedCsr)
}
