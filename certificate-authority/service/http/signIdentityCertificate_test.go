package http_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	certAuthURI "github.com/plgd-dev/hub/v2/certificate-authority/service/uri"
	"github.com/stretchr/testify/require"
)

func TestCertificateAuthorityServerSignIdentityCSR(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = "aa"
	csr, err := generateCertificate.GenerateCSR(cfg, priv)
	require.NoError(t, err)
	testSigningByFunction(t, func(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		var resp pb.SignCertificateResponse
		return &resp, httpDoSign(ctx, t, certAuthURI.SignIdentityCertificate, req, &resp)
	}, csr)
}

func TestCertificateAuthorityServerSignIdentityCSRWithEmptyCN(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	csr, err := generateCertificate.GenerateCSR(generateCertificate.Configuration{}, priv)
	require.NoError(t, err)
	testSigningByFunction(t, func(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		var resp pb.SignCertificateResponse
		return &resp, httpDoSign(ctx, t, certAuthURI.SignIdentityCertificate, req, &resp)
	}, csr)
}
