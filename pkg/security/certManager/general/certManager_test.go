package general_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/general"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
	"github.com/stretchr/testify/require"
)

func getCA(t *testing.T, validFrom time.Time, validFor time.Duration) ([]byte, *ecdsa.PrivateKey) {
	cfg := generateCertificate.Configuration{ValidFrom: validFrom.Format(time.RFC3339Nano), ValidFor: validFor}
	cfg.Subject.CommonName = "Test CA"
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caCrtPem, err := generateCertificate.GenerateRootCA(cfg, caKey)
	require.NoError(t, err)
	return caCrtPem, caKey
}

func getCert(t *testing.T, signerCA []byte, signerCAKey *ecdsa.PrivateKey, validFrom time.Time, validFor time.Duration) ([]byte, []byte) {
	signerCACerts, err := pkgX509.ParseX509(signerCA)
	require.NoError(t, err)
	cfg := generateCertificate.Configuration{ValidFrom: validFrom.Format(time.RFC3339Nano), ValidFor: validFor}
	cfg.Subject.CommonName = "Cert"
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	crtPem, err := generateCertificate.GenerateCert(cfg, key, signerCACerts, signerCAKey)
	require.NoError(t, err)
	b, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	return crtPem, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
}

func TestNew(t *testing.T) {
	// tmp dir
	tmpDir, err := os.MkdirTemp("/tmp", "test")
	require.NoError(t, err)
	defer func() {
		_ = deleteTmpDir(tmpDir)
	}()
	// ca
	caFile, err := os.CreateTemp(tmpDir, "ca")
	require.NoError(t, err)
	err = caFile.Close()
	require.NoError(t, err)

	crtFile, err := os.CreateTemp(tmpDir, "crt")
	require.NoError(t, err)
	err = crtFile.Close()
	require.NoError(t, err)

	keyFile, err := os.CreateTemp(tmpDir, "key")
	require.NoError(t, err)
	err = keyFile.Close()
	require.NoError(t, err)

	caPem, caKey := getCA(t, time.Now(), time.Second*100)
	crtPem, keyPem := getCert(t, caPem, caKey, time.Now(), time.Second*100)

	config := createTmpCertFiles(t, caFile.Name(), caPem, crtFile.Name(), crtPem, keyFile.Name(), keyPem)

	logger := log.NewLogger(log.MakeDefaultConfig())
	// cert manager
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		_ = fileWatcher.Close()
	}()
	mng, err := general.New(config, fileWatcher, logger)
	require.NoError(t, err)
	defer mng.Close()

	tlsConfig := mng.GetServerTLSConfig()
	require.NotNil(t, tlsConfig.GetCertificate)
	firstCrt, err := tlsConfig.GetCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, firstCrt)

	// delete crt/key files
	deleteTmpCertFiles(t, config)
	// create new crt/key files
	createTmpCertFiles(t, caFile.Name(), caPem, crtFile.Name(), crtPem, keyFile.Name(), keyPem)
	tlsConfig = mng.GetServerTLSConfig()
	require.NotNil(t, tlsConfig.GetCertificate)
	secondCrt, err := tlsConfig.GetCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, secondCrt)

	require.Equal(t, firstCrt.Certificate, secondCrt.Certificate)
}

func createTmpCertFiles(t *testing.T, caFile string, caPem []byte, crtFile string, crtPem []byte, keyFile string, keyPem []byte) general.Config {
	// ca
	err := os.WriteFile(caFile, caPem, os.FileMode(os.O_RDWR))
	require.NoError(t, err)

	// crt
	err = os.WriteFile(crtFile, crtPem, os.FileMode(os.O_RDWR))
	require.NoError(t, err)

	// key
	err = os.WriteFile(keyFile, keyPem, os.FileMode(os.O_RDWR))
	require.NoError(t, err)

	cfg := general.Config{
		CAPool:   []urischeme.URIScheme{urischeme.URIScheme(caFile)},
		KeyFile:  urischeme.URIScheme(keyFile),
		CertFile: urischeme.URIScheme(crtFile),
	}
	return cfg
}

func deleteTmpCertFiles(t *testing.T, cfg general.Config) {
	for _, ca := range cfg.CAPool {
		err := os.Remove(ca.FilePath())
		require.NoError(t, err)
	}
	err := os.Remove(cfg.CertFile.FilePath())
	require.NoError(t, err)
	err = os.Remove(cfg.KeyFile.FilePath())
	require.NoError(t, err)
}

func deleteTmpDir(tmpDir string) error {
	return os.RemoveAll(tmpDir)
}

// Check when ca expires
func TestCertManagerWithExpiredCA(t *testing.T) {
	// tmp dir
	tmpDir, err := os.MkdirTemp("/tmp", "test")
	require.NoError(t, err)
	defer func() {
		_ = deleteTmpDir(tmpDir)
	}()
	// ca
	caFile, err := os.CreateTemp(tmpDir, "ca")
	require.NoError(t, err)
	err = caFile.Close()
	require.NoError(t, err)

	crtFile, err := os.CreateTemp(tmpDir, "crt")
	require.NoError(t, err)
	err = crtFile.Close()
	require.NoError(t, err)

	keyFile, err := os.CreateTemp(tmpDir, "key")
	require.NoError(t, err)
	err = keyFile.Close()
	require.NoError(t, err)

	caPem, caKey := getCA(t, time.Now(), time.Second*2)
	crtPem, keyPem := getCert(t, caPem, caKey, time.Now(), time.Second*100)

	config := createTmpCertFiles(t, caFile.Name(), caPem, crtFile.Name(), crtPem, keyFile.Name(), keyPem)
	logger := log.NewLogger(log.MakeDefaultConfig())
	// cert manager
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		_ = fileWatcher.Close()
	}()
	mng, err := general.New(config, fileWatcher, logger)
	require.NoError(t, err)
	defer mng.Close()
	pool := mng.GetCertificateAuthorities()
	require.NotNil(t, pool)
	require.Len(t, pool.Subjects(), 1) //nolint:staticcheck
	time.Sleep(time.Second * 2)
	pool = mng.GetCertificateAuthorities()
	require.NotNil(t, pool)
	require.Empty(t, pool.Subjects()) //nolint:staticcheck
	caPem, _ = getCA(t, time.Now(), time.Second*100)
	err = os.WriteFile(caFile.Name(), caPem, os.FileMode(os.O_RDWR))
	require.NoError(t, err)
	time.Sleep(time.Second * 1)
	pool = mng.GetCertificateAuthorities()
	require.NotNil(t, pool)
	require.Len(t, pool.Subjects(), 1) //nolint:staticcheck
}

// Check when cert expires
func TestCertManagerWithExpiredCertificate(t *testing.T) {
	// tmp dir
	tmpDir, err := os.MkdirTemp("/tmp", "test")
	require.NoError(t, err)
	defer func() {
		_ = deleteTmpDir(tmpDir)
	}()
	// ca
	caFile, err := os.CreateTemp(tmpDir, "ca")
	require.NoError(t, err)
	err = caFile.Close()
	require.NoError(t, err)

	crtFile, err := os.CreateTemp(tmpDir, "crt")
	require.NoError(t, err)
	err = crtFile.Close()
	require.NoError(t, err)

	keyFile, err := os.CreateTemp(tmpDir, "key")
	require.NoError(t, err)
	err = keyFile.Close()
	require.NoError(t, err)

	caPem, caKey := getCA(t, time.Now(), time.Second*100)
	crtPem, keyPem := getCert(t, caPem, caKey, time.Now(), time.Second*2)

	config := createTmpCertFiles(t, caFile.Name(), caPem, crtFile.Name(), crtPem, keyFile.Name(), keyPem)
	logger := log.NewLogger(log.MakeDefaultConfig())
	// cert manager
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		_ = fileWatcher.Close()
	}()
	mng, err := general.New(config, fileWatcher, logger)
	require.NoError(t, err)
	defer mng.Close()

	// check client cert
	tlsCfg := mng.GetClientTLSConfig()
	require.NotNil(t, tlsCfg)
	cert, err := tlsCfg.GetClientCertificate(&tls.CertificateRequestInfo{})
	require.NoError(t, err)
	require.NotNil(t, cert)

	// check server cert
	tlsCfg = mng.GetServerTLSConfig()
	require.NotNil(t, tlsCfg)
	cert, err = tlsCfg.GetCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	require.NotNil(t, cert)

	time.Sleep(time.Second * 2)

	// client cert is expired
	tlsCfg = mng.GetClientTLSConfig()
	require.NotNil(t, tlsCfg)
	_, err = tlsCfg.GetClientCertificate(&tls.CertificateRequestInfo{})
	require.Error(t, err)

	// server cert is expired
	tlsCfg = mng.GetServerTLSConfig()
	require.NotNil(t, tlsCfg)
	_, err = tlsCfg.GetCertificate(&tls.ClientHelloInfo{})
	require.Error(t, err)

	crtPem, keyPem = getCert(t, caPem, caKey, time.Now(), time.Second*100)
	err = os.WriteFile(crtFile.Name(), crtPem, os.FileMode(os.O_RDWR))
	require.NoError(t, err)
	err = os.WriteFile(keyFile.Name(), keyPem, os.FileMode(os.O_RDWR))
	require.NoError(t, err)
	time.Sleep(time.Second * 1)

	// client cert is valid
	tlsCfg = mng.GetClientTLSConfig()
	require.NotNil(t, tlsCfg)
	cert, err = tlsCfg.GetClientCertificate(&tls.CertificateRequestInfo{})
	require.NoError(t, err)
	require.NotNil(t, cert)

	// server cert is valid
	tlsCfg = mng.GetServerTLSConfig()
	require.NotNil(t, tlsCfg)
	cert, err = tlsCfg.GetCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	require.NotNil(t, cert)
}
