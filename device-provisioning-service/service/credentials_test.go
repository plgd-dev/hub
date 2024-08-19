package service_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/schema/csr"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	caService "github.com/plgd-dev/hub/v2/certificate-authority/test"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/uri"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/security"
	"github.com/plgd-dev/kit/v2/security/generateCertificate"
	"github.com/stretchr/testify/require"
)

func setupTLSConfig(t *testing.T) *tls.Config {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	signerCert, err := security.LoadX509(os.Getenv("TEST_DPS_INTERMEDIATE_CA_CERT"))
	require.NoError(t, err)
	signerKey, err := security.LoadX509PrivateKey(os.Getenv("TEST_DPS_INTERMEDIATE_CA_KEY"))
	require.NoError(t, err)

	certData, err := generateCertificate.GenerateCert(generateCertificate.Configuration{
		ValidFrom:          time.Now().Add(-time.Hour).Format(time.RFC3339),
		ValidFor:           2 * time.Hour,
		ExtensionKeyUsages: []string{"client", "server"},
	}, priv, signerCert, signerKey)
	require.NoError(t, err)
	b, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	key := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	crt, err := tls.X509KeyPair(certData, key)
	require.NoError(t, err)
	caPool := x509.NewCertPool()
	for _, c := range signerCert {
		caPool.AddCert(c)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{
			crt,
		},
		RootCAs: caPool,
	}
}

func toCbor(t *testing.T, v interface{}) []byte {
	data, err := cbor.Encode(v)
	require.NoError(t, err)
	return data
}

func fromCbor(t *testing.T, w io.Reader, v interface{}) {
	err := cbor.ReadFrom(w, v)
	require.NoError(t, err)
}

func privateKeyToPem(t *testing.T, p *ecdsa.PrivateKey) []byte {
	b, err := x509.MarshalECPrivateKey(p)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
}

func TestCredentials(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId)
	defer hubShutdown()
	caCfg := caService.MakeConfig(t)
	caCfg.Signer.ExpiresIn = time.Hour * 10
	caShutdown := caService.New(t, caCfg)
	defer caShutdown()

	dpsCfg := test.MakeConfig(t)
	shutDown := test.New(t, dpsCfg)
	defer shutDown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	// create connection via mfg certificate
	c, err := tcp.Dial(dpsCfg.APIs.COAP.Addr, options.WithTLS(setupTLSConfig(t)), options.WithContext(ctx))
	require.NoError(t, err)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	deviceID, err := uuid.NewRandom()
	require.NoError(t, err)

	csrReq, err := generateCertificate.GenerateIdentityCSR(generateCertificate.Configuration{}, deviceID.String(), priv)
	require.NoError(t, err)

	req := service.CredentialsRequest{
		CSR: service.CSR{
			Encoding: csr.CertificateEncoding_PEM,
			Data:     string(csrReq),
		},
	}

	resp, err := c.Post(ctx, uri.Credentials, message.AppOcfCbor, bytes.NewReader(toCbor(t, req)))
	require.NoError(t, err)

	var creds credential.CredentialUpdateRequest
	fromCbor(t, resp.Body(), &creds)

	require.Len(t, creds.Credentials, 2)

	// create connection via identity certificate
	key := privateKeyToPem(t, priv)
	crt, err := tls.X509KeyPair(creds.Credentials[0].PublicData.Data(), key)
	require.NoError(t, err)
	caPool := x509.NewCertPool()
	signerCert, err := security.LoadX509(os.Getenv("TEST_ROOT_CA_CERT"))
	require.NoError(t, err)
	for _, c := range signerCert {
		caPool.AddCert(c)
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{
			crt,
		},
		RootCAs: caPool,
	}
	c1, err := tcp.Dial(dpsCfg.APIs.COAP.Addr, options.WithTLS(tlsCfg), options.WithContext(ctx))
	require.NoError(t, err)

	_, err = c1.Post(ctx, uri.Credentials, message.AppOcfCbor, bytes.NewReader(toCbor(t, req)))
	require.Error(t, err)

	_ = c1.Close()
}

const testPSK = "0123456789abcdef"

func TestCredentialsWithPSK(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId)
	defer hubShutdown()
	caCfg := caService.MakeConfig(t)
	caCfg.Signer.ExpiresIn = time.Hour * 10
	caShutdown := caService.New(t, caCfg)
	defer caShutdown()

	psk := testPSK
	pskFile := writeToTempFile(t, "psk.key", []byte(psk))
	defer func() {
		err := os.Remove(pskFile)
		require.NoError(t, err)
	}()

	dpsCfg := test.MakeConfig(t)
	dpsCfg.EnrollmentGroups[0].PreSharedKeyFile = urischeme.URIScheme(pskFile)
	shutDown := test.New(t, dpsCfg)
	defer shutDown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	// create connection via mfg certificate
	c, err := tcp.Dial(dpsCfg.APIs.COAP.Addr, options.WithTLS(setupTLSConfig(t)), options.WithContext(ctx))
	require.NoError(t, err)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	deviceID, err := uuid.NewRandom()
	require.NoError(t, err)

	csrReq, err := generateCertificate.GenerateIdentityCSR(generateCertificate.Configuration{}, deviceID.String(), priv)
	require.NoError(t, err)

	req := service.CredentialsRequest{
		CSR: service.CSR{
			Encoding: csr.CertificateEncoding_PEM,
			Data:     string(csrReq),
		},
	}

	resp, err := c.Post(ctx, uri.Credentials, message.AppOcfCbor, bytes.NewReader(toCbor(t, req)))
	require.NoError(t, err)

	var creds credential.CredentialUpdateRequest
	fromCbor(t, resp.Body(), &creds)

	require.Len(t, creds.Credentials, 3)
}
