package grpc_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/kit/v2/security/generateCertificate"
	"github.com/stretchr/testify/require"
)

func TestCertificateAuthorityServerSignIdentityCSR(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = "aa"
	csr, err := generateCertificate.GenerateCSR(cfg, priv)
	require.NoError(t, err)
	testSigningByFunction(t, func(ctx context.Context, c pb.CertificateAuthorityClient, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		return c.SignIdentityCertificate(ctx, req)
	}, csr)
}

func TestCertificateAuthorityServerSignIdentityCSRWithEmptyCN(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	csr, err := generateCertificate.GenerateCSR(generateCertificate.Configuration{}, priv)
	require.NoError(t, err)
	testSigningByFunction(t, func(ctx context.Context, c pb.CertificateAuthorityClient, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		return c.SignIdentityCertificate(ctx, req)
	}, csr)
}
