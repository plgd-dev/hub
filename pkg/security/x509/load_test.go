package x509_test

import (
	"os"
	"path/filepath"
	"testing"

	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
	testX509 "github.com/plgd-dev/hub/v2/test/security/x509"
	"github.com/stretchr/testify/require"
)

func TestReadX509(t *testing.T) {
	certPath := filepath.Join(os.TempDir(), "test-cert.pem")
	defer os.Remove(certPath)

	certPEM, _ := testX509.CreateCACertificate(t)
	err := os.WriteFile(certPath, certPEM, 0o600)
	require.NoError(t, err)

	certs, err := pkgX509.ReadX509(certPath)
	require.NoError(t, err)
	require.Len(t, certs, 1)
}

func TestReadPrivateKey(t *testing.T) {
	keyPath := filepath.Join(os.TempDir(), "test-key.pem")
	defer os.Remove(keyPath)

	_, key := testX509.CreateCACertificate(t)
	keyPEM := testX509.PrivateKeyToPem(t, key)
	err := os.WriteFile(keyPath, keyPEM, 0o600)
	require.NoError(t, err)

	key, err = pkgX509.ReadPrivateKey(keyPath)
	require.NoError(t, err)
	require.NotNil(t, key)
}
