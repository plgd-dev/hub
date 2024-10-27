package grpc_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	"github.com/plgd-dev/hub/v2/certificate-authority/test"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func createCACertificate(t *testing.T) ([]byte, *ecdsa.PrivateKey) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = "rootCA"
	cfg.ValidFor = time.Hour * 24
	cfg.BasicConstraints.MaxPathLen = 1000
	rootCA, err := generateCertificate.GenerateRootCA(cfg, priv)
	require.NoError(t, err)
	return rootCA, priv
}

func privateKeyToPem(t *testing.T, priv *ecdsa.PrivateKey) []byte {
	privKey, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privKey})
}

func TestReloadCerts(t *testing.T) {
	const ownerClaim = "sub"
	store, closeStore := test.NewMongoStore(t)
	defer closeStore()

	logger := log.NewLogger(log.MakeDefaultConfig())

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		err = fileWatcher.Close()
		require.NoError(t, err)
	}()

	crt, err := os.CreateTemp("", "TestReloadCerts***.crt")
	require.NoError(t, err)
	defer func() {
		err = os.Remove(crt.Name())
		require.NoError(t, err)
	}()
	err = crt.Close()
	require.NoError(t, err)

	key, err := os.CreateTemp("", "TestReloadCerts***.key")
	require.NoError(t, err)
	defer func() {
		err = os.Remove(key.Name())
		require.NoError(t, err)
	}()
	err = key.Close()
	require.NoError(t, err)

	crtPem, privKey := createCACertificate(t)
	err = os.WriteFile(crt.Name(), crtPem, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(key.Name(), privateKeyToPem(t, privKey), 0o600)
	require.NoError(t, err)

	s := test.MakeConfig(t).Signer
	s.CAPool = []string{crt.Name()}
	s.CertFile = urischeme.URIScheme(crt.Name())
	s.KeyFile = urischeme.URIScheme(key.Name())
	err = s.Validate()
	require.NoError(t, err)

	ca, err := grpc.NewCertificateAuthorityServer(ownerClaim, config.HubID(), "https://"+config.CERTIFICATE_AUTHORITY_HTTP_HOST, s, store, fileWatcher, logger)
	require.NoError(t, err)
	defer ca.Close()

	// test reload certs with the different certs

	s1 := ca.GetSigner()

	crtPem, privKey = createCACertificate(t)
	err = os.WriteFile(crt.Name(), crtPem, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(key.Name(), privateKeyToPem(t, privKey), 0o600)
	require.NoError(t, err)

	// wait for reload certs
	time.Sleep(time.Second)

	s2 := ca.GetSigner()
	require.NotEqual(t, s1, s2)

	// test reload certs with the same certs

	err = os.WriteFile(crt.Name(), crtPem, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(key.Name(), privateKeyToPem(t, privKey), 0o600)
	require.NoError(t, err)

	// wait for reload certs
	time.Sleep(time.Second)

	s3 := ca.GetSigner()
	require.Equal(t, s2, s3)
}
