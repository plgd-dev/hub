package service_test

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	pbCA "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	caTest "github.com/plgd-dev/hub/v2/certificate-authority/test"
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

func getSigningRecords(ctx context.Context, t *testing.T, addr string, certificates []tls.Certificate) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(
		credentials.NewTLS(&tls.Config{
			RootCAs:      test.GetRootCertificatePool(t),
			Certificates: certificates,
		})),
	)
	require.NoError(t, err)
	defer conn.Close()
	caClient := pbCA.NewCertificateAuthorityClient(conn)
	cl, err := caClient.GetSigningRecords(ctx, &pbCA.GetSigningRecordsRequest{})
	if err == nil {
		_ = cl.CloseSend()
		for {
			_, err2 := cl.Recv()
			if err2 != nil {
				break
			}
		}
	}
	return err
}

func getNewCertificate(ctx context.Context, t *testing.T, addr string, pkI crypto.PrivateKey, certificates []tls.Certificate) ([]byte, error) {
	pk := pkI.(*ecdsa.PrivateKey)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(
		credentials.NewTLS(&tls.Config{
			RootCAs:      test.GetRootCertificatePool(t),
			Certificates: certificates,
		})),
	)
	require.NoError(t, err)
	caClient := pbCA.NewCertificateAuthorityClient(conn)

	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = "test"
	csr, err := generateCertificate.GenerateCSR(cfg, pk)
	require.NoError(t, err)

	resp, err := caClient.SignCertificate(ctx, &pbCA.SignCertificateRequest{CertificateSigningRequest: csr})
	if err != nil {
		return nil, err
	}
	return resp.GetCertificate(), nil
}

func marshalPrivateKey(t *testing.T, pk crypto.PrivateKey) []byte {
	b, err := x509.MarshalECPrivateKey(pk.(*ecdsa.PrivateKey))
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
}

func getNewTLSCertificate(ctx context.Context, t *testing.T, addr string, certificates []tls.Certificate) tls.Certificate {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	// get certificate - insecure
	certData1, err := getNewCertificate(ctx, t, config.CERTIFICATE_AUTHORITY_HOST, pk, nil)
	require.NoError(t, err)
	crt, err := tls.X509KeyPair(certData1, marshalPrivateKey(t, pk))
	require.NoError(t, err)
	return crt
}

func TestGetSigningRecords(t *testing.T) {
	if config.ACTIVE_DATABASE() == database.CqlDB {
		t.Skip("revocation list not supported for CqlDB")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	shutdown := testService.SetUpServices(ctx, t, testService.SetUpServicesOAuth|testService.SetUpServicesMachine2MachineOAuth)
	defer shutdown()

	ctx = pkgGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	// start insecure ca
	caCfg := caTest.MakeConfig(t)
	// CRL list should be valid for 10 sec after it is issued
	caCfg.Signer.CRL.Enabled = false
	caCfg.Signer.CRL.ExpiresIn = time.Hour
	err := caCfg.Validate()
	require.NoError(t, err)
	caShutdown := caTest.New(t, caCfg)
	crtWithoutCrlDistributionPoint := getNewTLSCertificate(ctx, t, config.CERTIFICATE_AUTHORITY_HOST, nil)
	caShutdown()
	caCfg.Signer.CRL.Enabled = true
	caShutdown = caTest.New(t, caCfg)
	crtWithCrlDistributionPoint := getNewTLSCertificate(ctx, t, config.CERTIFICATE_AUTHORITY_HOST, nil)
	caShutdown()

	// start secure ca
	caCfg.APIs.GRPC.TLS.ClientCertificateRequired = true
	caCfg.APIs.GRPC.TLS.CRL.Enabled = true
	httpClientConfig := config.MakeHttpClientConfig()
	caCfg.APIs.GRPC.TLS.CRL.HTTP = &httpClientConfig
	caCfg.Signer.CRL.Enabled = false // generate cert without distribution point
	err = caCfg.Validate()
	require.NoError(t, err)
	caShutdown = caTest.New(t, caCfg)
	defer caShutdown()

	// second ca on different port
	const ca_addr = "localhost:30011"
	caCfg2 := caTest.MakeConfig(t)
	caCfg2.APIs.GRPC = config.MakeGrpcServerConfig(ca_addr)
	caCfg2.APIs.GRPC.TLS.ClientCertificateRequired = true
	caCfg2.APIs.GRPC.TLS.CRL.Enabled = true
	caCfg2.APIs.GRPC.TLS.CRL.HTTP = &httpClientConfig
	caCfg2.APIs.HTTP.ExternalAddress = "https://localhost:30012"
	caCfg2.APIs.HTTP.Addr = "localhost:30012"
	err = caCfg2.Validate()
	require.NoError(t, err)
	caShutdown2 := caTest.New(t, caCfg2)
	defer caShutdown2()

	err = getSigningRecords(ctx, t, ca_addr, []tls.Certificate{crtWithCrlDistributionPoint})
	require.Error(t, err)

	certData, err := getNewCertificate(ctx, t, config.CERTIFICATE_AUTHORITY_HOST, crtWithoutCrlDistributionPoint.PrivateKey, []tls.Certificate{crtWithoutCrlDistributionPoint})
	require.NoError(t, err)

	crtWithoutCrlDistributionPoint, err = tls.X509KeyPair(certData, marshalPrivateKey(t, crtWithoutCrlDistributionPoint.PrivateKey))
	require.NoError(t, err)

	certData, err = getNewCertificate(ctx, t, ca_addr, crtWithCrlDistributionPoint.PrivateKey, []tls.Certificate{crtWithoutCrlDistributionPoint})
	require.NoError(t, err)

	crtWithCrlDistributionPoint, err = tls.X509KeyPair(certData, marshalPrivateKey(t, crtWithCrlDistributionPoint.PrivateKey))
	require.NoError(t, err)

	// use new crtWithCrlDistributionPoint
	err = getSigningRecords(ctx, t, ca_addr, []tls.Certificate{crtWithCrlDistributionPoint})
	require.NoError(t, err)

	err = getSigningRecords(ctx, t, config.CERTIFICATE_AUTHORITY_HOST, []tls.Certificate{crtWithCrlDistributionPoint})
	require.NoError(t, err)

	revokedCertificate := crtWithCrlDistributionPoint
	// use crt2 without distribution point
	_, err = getNewCertificate(ctx, t, config.CERTIFICATE_AUTHORITY_HOST, crtWithCrlDistributionPoint.PrivateKey, []tls.Certificate{crtWithoutCrlDistributionPoint})
	require.NoError(t, err)

	// try use revoked crtWithCrlDistributionPoint
	err = getSigningRecords(ctx, t, ca_addr, []tls.Certificate{revokedCertificate})
	require.Error(t, err)
}
