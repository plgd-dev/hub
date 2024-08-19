package service_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	caService "github.com/plgd-dev/hub/v2/certificate-authority/test"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/security/generateCertificate"
	"github.com/stretchr/testify/require"
)

func TestWithUntrustedCertificate(t *testing.T) {
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

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	cfg := generateCertificate.Configuration{
		ValidFrom:          time.Now().Add(-time.Hour).Format(time.RFC3339),
		ValidFor:           2 * time.Hour,
		ExtensionKeyUsages: []string{"client", "server"},
	}
	cfg.Subject.CommonName = "test"
	data, err := generateCertificate.GenerateRootCA(cfg, priv)
	require.NoError(t, err)
	b, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	key := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	crt, err := tls.X509KeyPair(data, key)
	require.NoError(t, err)

	// create connection via mfg certificate
	c, err := tcp.Dial(dpsCfg.APIs.COAP.Addr, options.WithTLS(&tls.Config{
		Certificates: []tls.Certificate{
			crt,
		},
		InsecureSkipVerify: true,
	}), options.WithContext(ctx))
	require.NoError(t, err)
	err = c.Ping(ctx)
	require.Error(t, err)
}
