package grpc_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func createIdentityCSR(t *testing.T, deviceID string) []byte {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = "uuid:" + deviceID
	csr, err := generateCertificate.GenerateCSR(cfg, priv)
	require.NoError(t, err)
	return csr
}

func TestCertificateAuthorityServerSignIdentityCSR(t *testing.T) {
	csr := createIdentityCSR(t, "da040434-ca75-490d-8723-0311ca05f118")
	testSigningByFunction(t, func(ctx context.Context, c pb.CertificateAuthorityClient, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		return c.SignIdentityCertificate(ctx, req)
	}, csr, csr)
}

func TestCertificateAuthorityServerSignDeviceIdentityCSRWithDifferentPublicKeys(t *testing.T) {
	csr := createIdentityCSR(t, "da040434-ca75-490d-8723-0311ca05f118")
	csr1 := createIdentityCSR(t, "da040434-ca75-490d-8723-0311ca05f118")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = pkgGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.CERTIFICATE_AUTHORITY_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewCertificateAuthorityClient(conn)

	_, err = c.SignIdentityCertificate(ctx, &pb.SignCertificateRequest{CertificateSigningRequest: csr})
	require.NoError(t, err)

	_, err = c.SignIdentityCertificate(ctx, &pb.SignCertificateRequest{CertificateSigningRequest: csr1})
	require.Error(t, err)
}

func TestCertificateAuthorityServerSignOwnerIdentityCSRWithDifferentPublicKeys(t *testing.T) {
	owner := events.OwnerToUUID(oauthService.DeviceUserID)

	csr := createIdentityCSR(t, owner)
	csr1 := createIdentityCSR(t, owner)

	require.NotEqual(t, csr, csr1)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = pkgGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.CERTIFICATE_AUTHORITY_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewCertificateAuthorityClient(conn)

	_, err = c.SignIdentityCertificate(ctx, &pb.SignCertificateRequest{CertificateSigningRequest: csr})
	require.NoError(t, err)

	_, err = c.SignIdentityCertificate(ctx, &pb.SignCertificateRequest{CertificateSigningRequest: csr1})
	require.NoError(t, err)
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
