//go:build test
// +build test

package service_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"testing"
	"time"

	pbCA "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	caTest "github.com/plgd-dev/hub/v2/certificate-authority/test"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestCertificateWithCRL(t *testing.T) {
	if config.ACTIVE_DATABASE() == database.CqlDB {
		t.Skip("revocation list not supported for CqlDB")
	}
	caCfg := caTest.MakeConfig(t)
	// CRL list should be valid for 10 sec after it is issued
	caCfg.Signer.CRL.Enabled = true
	caCfg.Signer.CRL.ExpiresIn = time.Second * 10
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.APIs.COAP.TLS.Enabled = new(bool)
	*coapgwCfg.APIs.COAP.TLS.Enabled = true
	coapgwCfg.APIs.COAP.TLS.Embedded.ClientCertificateRequired = true
	shutdown := setUp(t, testService.WithCAConfig(caCfg), testService.WithCOAPGWConfig(coapgwCfg))
	defer shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	tokenWithoutDeviceID := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, tokenWithoutDeviceID)
	conn, err := grpc.NewClient(config.CERTIFICATE_AUTHORITY_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	caClient := pbCA.NewCertificateAuthorityClient(conn)

	signerKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	cg := coapgwTest.NewCACertificateGenerator(caClient, signerKey)

	crt, err := cg.GetIdentityCertificate(ctx, CertIdentity)
	require.NoError(t, err)
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{crt},
		InsecureSkipVerify: true,
	}
	co := testCoapDialWithHandler(t, makeTestCoapHandler(t), WithTLSConfig(tlsConfig))
	require.NotEmpty(t, co)
	testSignUp(t, CertIdentity, co)
	_ = co.Close()

	// revoke all certs for device
	resp, err := caClient.DeleteSigningRecords(ctx, &pbCA.DeleteSigningRecordsRequest{
		DeviceIdFilter: []string{CertIdentity},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.Count)

	time.Sleep(time.Second * 10) // wait for CRL to expire in cache
	// sign-up with revoked cert should fail
	co = testCoapDialWithHandler(t, makeTestCoapHandler(t), WithTLSConfig(tlsConfig))
	require.NotEmpty(t, co)
	_, err = doSignUp(t, CertIdentity, co)
	_ = co.Close()
	require.Error(t, err)
}
