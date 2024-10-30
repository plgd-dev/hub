package test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
)

type LocalCertificateGenerator struct {
	signerCACertificate []*x509.Certificate
	signerCAKey         *ecdsa.PrivateKey
}

func NewLocalCertificateGenerator(sc []*x509.Certificate, sk *ecdsa.PrivateKey) *LocalCertificateGenerator {
	return &LocalCertificateGenerator{
		signerCACertificate: sc,
		signerCAKey:         sk,
	}
}

func getTLSCertificate(certPEMBlock []byte, pk *ecdsa.PrivateKey) (tls.Certificate, error) {
	b, err := x509.MarshalECPrivateKey(pk)
	if err != nil {
		return tls.Certificate{}, err
	}
	key := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	crt, err := tls.X509KeyPair(certPEMBlock, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return crt, nil
}

func (g *LocalCertificateGenerator) getCertificate(identityCert bool, deviceID string, validTo time.Time) (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	var certData []byte
	if identityCert {
		certData, err = generateCertificate.GenerateIdentityCert(generateCertificate.Configuration{
			ValidFrom: time.Now().Add(-time.Hour).Format(time.RFC3339),
			ValidFor:  time.Until(validTo) + time.Hour,
		}, deviceID, priv, g.signerCACertificate, g.signerCAKey)
		if err != nil {
			return tls.Certificate{}, err
		}
	} else {
		c := generateCertificate.Configuration{
			ValidFrom: time.Now().Add(-time.Hour).Format(time.RFC3339),
			ValidFor:  time.Until(validTo) + time.Hour,
		}
		c.Subject.CommonName = "non-identity-cert"
		c.ExtensionKeyUsages = []string{"client", "server"}
		certData, err = generateCertificate.GenerateCert(c, priv, g.signerCACertificate, g.signerCAKey)
		if err != nil {
			return tls.Certificate{}, err
		}
	}
	return getTLSCertificate(certData, priv)
}

func (g *LocalCertificateGenerator) GetIdentityCertificate(deviceID string, validTo time.Time) (tls.Certificate, error) {
	return g.getCertificate(true, deviceID, validTo)
}

func (g *LocalCertificateGenerator) GetCertificate(validTo time.Time) (tls.Certificate, error) {
	return g.getCertificate(false, "", validTo)
}

type CACertificateGenerator struct {
	caClient  pb.CertificateAuthorityClient
	signerKey *ecdsa.PrivateKey
}

func NewCACertificateGenerator(caClient pb.CertificateAuthorityClient, signerKey *ecdsa.PrivateKey) *CACertificateGenerator {
	return &CACertificateGenerator{
		caClient:  caClient,
		signerKey: signerKey,
	}
}

func (c *CACertificateGenerator) GetIdentityCertificate(ctx context.Context, deviceID string) (tls.Certificate, error) {
	csr, err := generateCertificate.GenerateIdentityCSR(generateCertificate.Configuration{}, deviceID, c.signerKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("cannot generate identity csr: %w", err)
	}

	resp, err := c.caClient.SignIdentityCertificate(ctx, &pb.SignCertificateRequest{
		CertificateSigningRequest: csr,
	})
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("certificate authority failed to sign certificate: %w", err)
	}
	return getTLSCertificate(resp.GetCertificate(), c.signerKey)
}
