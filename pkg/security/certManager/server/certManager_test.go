package server_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/plgd-dev/cloud/pkg/security/certManager/server"
	"github.com/plgd-dev/kit/log"
	"github.com/stretchr/testify/require"
)

var TestCaCrt = `-----BEGIN CERTIFICATE-----
MIIBvzCCAWSgAwIBAgIRAKhVk049hVtC24ohZqzXSHAwCgYIKoZIzj0EAwIwTjEN
MAsGA1UEBhMEVGVzdDENMAsGA1UEBxMEVGVzdDENMAsGA1UEChMEVGVzdDENMAsG
A1UECxMEVGVzdDEQMA4GA1UEAxMHVGVzdCBDQTAeFw0yMDAyMDYxMTA1NTRaFw0z
MDAyMDMxMTA1NTRaME4xDTALBgNVBAYTBFRlc3QxDTALBgNVBAcTBFRlc3QxDTAL
BgNVBAoTBFRlc3QxDTALBgNVBAsTBFRlc3QxEDAOBgNVBAMTB1Rlc3QgQ0EwWTAT
BgcqhkjOPQIBBggqhkjOPQMBBwNCAAQ1JZwVjcOn0qxLr1rCQN5cYBdePoV+i2ie
ri+6dRt8JEqpR1+694+yWllCu+ldTlYVduU/pUOrUJ4oyYU3c6floyMwITAOBgNV
HQ8BAf8EBAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAKBggqhkjOPQQDAgNJADBGAiEA
2xvxZ7EYxhUusLpZiKJmzKg2CZCAP4v8uzlI1JqePqACIQDJQlUwrVdARpC02v+J
3CcezG3lWHuMG1sTW4zekKuFiA==
-----END CERTIFICATE-----
`

var TestCrt = `-----BEGIN CERTIFICATE-----
MIIB2jCCAYGgAwIBAgIRAP5nV3phj3WbAHFiT/cY7vwwCgYIKoZIzj0EAwIwTjEN
MAsGA1UEBhMEVGVzdDENMAsGA1UEBxMEVGVzdDENMAsGA1UEChMEVGVzdDENMAsG
A1UECxMEVGVzdDEQMA4GA1UEAxMHVGVzdCBDQTAeFw0yMDAyMDYxMTA2MzZaFw0z
MDAyMDMxMTA2MzZaMC0xDTALBgNVBAYTBFRlc3QxDTALBgNVBAoTBFRlc3QxDTAL
BgNVBAMTBHRlc3QwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQn+5ei51r7pUNt
VKfn2rRsUsLROk0rDOQG9oEvzqjARiZwGEEumSkCdDV5MYpMYt0BmxX42dk8vXue
K3VxuI3ao2EwXzAjBgNVHREEHDAaggR0ZXN0ggxodHRwczovL3Rlc3SHBH8AAAEw
DAYDVR0TBAUwAwEBADALBgNVHQ8EBAMCA4gwHQYDVR0lBBYwFAYIKwYBBQUHAwIG
CCsGAQUFBwMBMAoGCCqGSM49BAMCA0cAMEQCIAOm/45P8C/njZZrs8iYEotOk3oQ
f7d8FwSKAagbNWomAiABQBEb9CvfG3so04yKmIMd/2XB5LXM2SQfBKdg/nMD8A==
-----END CERTIFICATE-----
`
var TestCrtKey = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIAqNQjvFqI95fIE/2UOMBM+mOJq0mCCkZTj/clWsa5VCoAoGCCqGSM49
AwEHoUQDQgAEJ/uXouda+6VDbVSn59q0bFLC0TpNKwzkBvaBL86owEYmcBhBLpkp
AnQ1eTGKTGLdAZsV+NnZPL17nit1cbiN2g==
-----END EC PRIVATE KEY-----
`

func TestNew(t *testing.T) {
	//tmp dir
	tmpDir, err := ioutil.TempDir("/tmp", "test")
	defer deleteTmpDir(tmpDir)
	//ca
	caFile, err := ioutil.TempFile(tmpDir, "ca")
	caFile.Close()

	crtFile, err := ioutil.TempFile(tmpDir, "crt")
	crtFile.Close()

	keyFile, err := ioutil.TempFile(tmpDir, "key")
	keyFile.Close()

	config := createTmpCertFiles(t, caFile.Name(), crtFile.Name(), keyFile.Name())

	logger, err := log.NewLogger(log.Config{})
	require.NoError(t, err)
	//cert manager
	mng, err := server.New(config, logger)
	require.NoError(t, err)

	tlsConfig := mng.GetTLSConfig()
	require.NotNil(t, tlsConfig.GetCertificate)
	firstCrt, err := tlsConfig.GetCertificate(nil)
	require.NotNil(t, firstCrt)

	//delete crt/key files
	deleteTmpCertFiles(t, config)
	//create new crt/key files
	config = createTmpCertFiles(t, caFile.Name(), crtFile.Name(), keyFile.Name())
	tlsConfig = mng.GetTLSConfig()

	require.NotNil(t, tlsConfig.GetCertificate)
	secondCrt, err := tlsConfig.GetCertificate(nil)
	require.NotNil(t, secondCrt)

	require.Equal(t, firstCrt.Certificate, secondCrt.Certificate)

}

func createTmpCertFiles(t *testing.T, caFile, crtFile, keyFile string) server.Config {
	//ca
	err := ioutil.WriteFile(caFile, []byte(TestCaCrt), os.FileMode(os.O_RDWR))
	require.NoError(t, err)

	// crt
	err = ioutil.WriteFile(crtFile, []byte(TestCrt), os.FileMode(os.O_RDWR))
	require.NoError(t, err)

	//key
	err = ioutil.WriteFile(keyFile, []byte(TestCrtKey), os.FileMode(os.O_RDWR))
	require.NoError(t, err)

	cfg := server.Config{
		CAPool:   caFile,
		KeyFile:  keyFile,
		CertFile: crtFile,
	}
	return cfg
}

func deleteTmpCertFiles(t *testing.T, cfg server.Config) {
	err := os.Remove(cfg.CAPool)
	require.NoError(t, err)
	err = os.Remove(cfg.CertFile)
	require.NoError(t, err)
	err = os.Remove(cfg.KeyFile)
	require.NoError(t, err)
}

func deleteTmpDir(tmpDir string) {
	os.RemoveAll(tmpDir)
}
