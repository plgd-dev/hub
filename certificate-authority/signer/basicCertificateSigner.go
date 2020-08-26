package signer

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

type BasicCertificateSigner struct {
	caCert         []*x509.Certificate
	caKey          crypto.PrivateKey
	validNotBefore time.Time
	validNotAfter  time.Time
}

func NewBasicCertificateSigner(caCert []*x509.Certificate, caKey crypto.PrivateKey, validNotBefore time.Time, validNotAfter time.Time) *BasicCertificateSigner {
	return &BasicCertificateSigner{caCert: caCert, caKey: caKey, validNotBefore: validNotBefore, validNotAfter: validNotAfter}
}

func createPemChain(intermedateCAs []*x509.Certificate, cert []byte) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 2048))

	// encode cert
	err := pem.Encode(buf, &pem.Block{
		Type: "CERTIFICATE", Bytes: cert,
	})
	if err != nil {
		return nil, err
	}
	// encode intermediates
	for _, ca := range intermedateCAs {
		if bytes.Equal(ca.RawIssuer, ca.RawSubject) {
			continue
		}
		err := pem.Encode(buf, &pem.Block{
			Type: "CERTIFICATE", Bytes: ca.Raw,
		})
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (s *BasicCertificateSigner) Sign(ctx context.Context, csr []byte) (signedCsr []byte, err error) {
	csrBlock, _ := pem.Decode(csr)
	if csrBlock == nil {
		err = fmt.Errorf("pem not found")
		return
	}

	certificateRequest, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		return nil, err
	}

	err = certificateRequest.CheckSignature()
	if err != nil {
		return nil, err
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
		DNSNames:           certificateRequest.DNSNames,
		IPAddresses:        certificateRequest.IPAddresses,
		URIs:               certificateRequest.URIs,
		EmailAddresses:     certificateRequest.EmailAddresses,
		ExtraExtensions:    certificateRequest.Extensions,
	}
	if len(s.caCert) == 0 {
		return nil, fmt.Errorf("cannot sign with empty signer CA certificates")
	}
	signedCsr, err = x509.CreateCertificate(rand.Reader, &template, s.caCert[0], certificateRequest.PublicKey, s.caKey)
	if err != nil {
		return nil, err
	}
	return createPemChain(s.caCert, signedCsr)
}
