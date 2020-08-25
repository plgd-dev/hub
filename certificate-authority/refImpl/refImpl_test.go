package refImpl

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	var config Config
	dir, err := ioutil.TempDir("", "gotesttmp")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	testSignerCerts(t, dir)
	os.Setenv("VALID_FROM", "now-1d")
	err = envconfig.Process("", &config)
	require.NoError(t, err)

	got, err := Init(config)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func testSignerCerts(t *testing.T, dir string) {
	crt := filepath.Join(dir, "cert.crt")
	if err := ioutil.WriteFile(crt, IdentityIntermediateCA, 0600); err != nil {
		assert.NoError(t, err)
	}
	crtKey := filepath.Join(dir, "cert.key")
	if err := ioutil.WriteFile(crtKey, IdentityIntermediateCAKey, 0600); err != nil {
		assert.NoError(t, err)
	}
	os.Setenv("SIGNER_CERTIFICATE", crt)
	os.Setenv("SIGNER_PRIVATE_KEY", crtKey)
}

var (
	IdentityIntermediateCA = []byte(`-----BEGIN CERTIFICATE-----
MIIBczCCARmgAwIBAgIRANntjEpzu9krzL0EG6fcqqgwCgYIKoZIzj0EAwIwETEP
MA0GA1UEAxMGUm9vdENBMCAXDTE5MDcxOTIwMzczOVoYDzIxMTkwNjI1MjAzNzM5
WjAZMRcwFQYDVQQDEw5JbnRlcm1lZGlhdGVDQTBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABKw1/6WHFcWtw67hH5DzoZvHgA0suC6IYLKms4IP/pds9wU320eDaENo
5860TOyKrGn7vW/cj/OVe2Dzr4KSFVijSDBGMA4GA1UdDwEB/wQEAwIBBjATBgNV
HSUEDDAKBggrBgEFBQcDATASBgNVHRMBAf8ECDAGAQH/AgEAMAsGA1UdEQQEMAKC
ADAKBggqhkjOPQQDAgNIADBFAiEAgPtnYpgwxmPhN0Mo8VX582RORnhcdSHMzFjh
P/li1WwCIFVVWBOrfBnTt7A6UfjP3ljAyHrJERlMauQR+tkD/aqm
-----END CERTIFICATE-----
`)
	IdentityIntermediateCAKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIPF4DPvFeiRL1G0ROd6MosoUGnvIG/2YxH0CbHwnLKxqoAoGCCqGSM49
AwEHoUQDQgAErDX/pYcVxa3DruEfkPOhm8eADSy4Lohgsqazgg/+l2z3BTfbR4No
Q2jnzrRM7Iqsafu9b9yP85V7YPOvgpIVWA==
-----END EC PRIVATE KEY-----
`)
)
