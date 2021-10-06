package jwt_test

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/v2/pkg/log"
	"github.com/plgd-dev/cloud/v2/pkg/security/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator(t *testing.T) {
	server := newTestJwks()
	defer server.Close()

	v := jwt.NewValidator(server.URL+uri, &noTLS)
	var c testClaims
	err := v.ParseWithClaims(token, &c)
	require.NoError(t, err)

	assert.Equal(t, "test.client.id", c.ClientID)
	assert.Equal(t, "user@example.com", c.Email)
	assert.Contains(t, c.Scope, "test.scope")
}

func TestClaims(t *testing.T) {
	server := newTestJwks()
	defer server.Close()

	v := jwt.NewValidator(server.URL+uri, &noTLS)
	var c jwt.Claims
	err := v.ParseWithClaims(token, &c)
	require.Error(t, err)
	require.Contains(t, err.Error(), "token is expired")

	assert.Equal(t, "test.client.id", c.ClientID())
	assert.Equal(t, "user@example.com", c.Email())
	assert.Contains(t, c.Scope(), "test.scope")

	assert.Contains(t, c.Audience(), "http://identity-server:3001/resources")
	assert.Contains(t, c.Audience(), "test.resource")
	exp, err := c.ExpiresAt()
	assert.NoError(t, err)
	assert.Equal(t, 2019, exp.Year())
	assert.Empty(t, c.ID())
	iat, err := c.IssuedAt()
	assert.NoError(t, err)
	assert.Equal(t, time.Time{}, iat)
	assert.Equal(t, c.Issuer(), "http://identity-server:3001")
	nbf, err := c.NotBefore()
	assert.NoError(t, err)
	assert.Equal(t, 2019, nbf.Year())
	assert.Equal(t, "1b87effa-34e2-4a44-82c6-6e0ab80209ff", c.Subject())
}

func TestParser(t *testing.T) {
	server := newTestJwks()
	defer server.Close()

	v := jwt.NewValidator(server.URL+uri, &noTLS)
	c, err := v.Parse(token)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Token is expired")

	assert.Equal(t, "test.client.id", c[jwt.ClaimClientID])
	assert.Equal(t, "user@example.com", c[jwt.ClaimEmail])
	assert.Contains(t, c[jwt.ClaimScope], "test.scope")

	assert.Equal(t, "local", c["idp"])
	assert.Contains(t, c["amr"], "pwd")
	assert.Contains(t, c[jwt.ClaimAudience], "http://identity-server:3001/resources")
	assert.Contains(t, c[jwt.ClaimAudience], "test.resource")
	assert.Equal(t, "http://identity-server:3001", c[jwt.ClaimIssuer])
	assert.Equal(t, "1b87effa-34e2-4a44-82c6-6e0ab80209ff", c[jwt.ClaimSubject])

	assert.Equal(t, 2019, time.Unix(int64(c[jwt.ClaimNotBefore].(float64)), 0).Year())
	assert.Equal(t, 2019, time.Unix(int64(c[jwt.ClaimExpiresAt].(float64)), 0).Year())
	assert.Equal(t, 2019, time.Unix(int64(c["auth_time"].(float64)), 0).Year())
}

func TestEmptyToken(t *testing.T) {
	server := newTestJwks()
	defer server.Close()

	v := jwt.NewValidator(server.URL+uri, &noTLS)
	_, err := v.Parse("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing token")

	var c jwt.Claims
	err = v.ParseWithClaims("", &c)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing token")
}

func TestInvalidToken(t *testing.T) {
	server := newTestJwks()
	defer server.Close()

	v := jwt.NewValidator(server.URL+uri, &noTLS)
	_, err := v.Parse("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not parse token")

	var c jwt.Claims
	err = v.ParseWithClaims("invalid", &c)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not parse token")
}

func newTestJwks() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(uri, func(w http.ResponseWriter, r *http.Request) {
		_, err := io.WriteString(w, jwks)
		log.Debugf("failed to write jwks: %v", err)
	})
	return httptest.NewServer(mux)
}

type testClaims struct {
	ClientID string   `json:"client_id"`
	Email    string   `json:"email"`
	Scope    []string `json:"scope"`
}

func (c testClaims) Valid() error {
	return nil
}

const token = `eyJhbGciOiJSUzI1NiIsImtpZCI6IjdmNzM1NWJmYmY0ZjVmOTNkZjJiZjg3OWE3OGUyMjNhIiwidHlwIjoiSldUIn0.eyJuYmYiOjE1NjM4ODk5NTgsImV4cCI6MTU2Mzg5MzU1OCwiaXNzIjoiaHR0cDovL2lkZW50aXR5LXNlcnZlcjozMDAxIiwiYXVkIjpbImh0dHA6Ly9pZGVudGl0eS1zZXJ2ZXI6MzAwMS9yZXNvdXJjZXMiLCJ0ZXN0LnJlc291cmNlIl0sImNsaWVudF9pZCI6InRlc3QuY2xpZW50LmlkIiwic3ViIjoiMWI4N2VmZmEtMzRlMi00YTQ0LTgyYzYtNmUwYWI4MDIwOWZmIiwiYXV0aF90aW1lIjoxNTYzODg5ODQ1LCJpZHAiOiJsb2NhbCIsImVtYWlsIjoidXNlckBleGFtcGxlLmNvbSIsInRlbmFudF9pZCI6IjdlYWE1MzJhLWMyYmYtNDRiNC05MTA5LWE3ODBmNGY2MzI2NCIsImRlZmF1bHRfcHJvZHVjdF91cmkiOiIvIiwicm9sZXMiOlsidGVzdC5yb2xlMSIsInRlc3Qucm9sZTIiXSwic2NvcGUiOlsidGVzdC5zY29wZSJdLCJhbXIiOlsicHdkIl19.jwgEJpn9aYZrWFzMRvW9ABZpA_MnZDNZcfWJtFm-luyYBm2D06P6bsKTH0mYcxKTVSVHnEczRecQumwtWjyvyVVtCLDOW1GVPseChw-By152QxOIeOpsJh2zPEKlVaqvPkFKOfFGPWYb99RN8-Cfv9hKYtUCrFhHCwcWcrzCngiYElGvRkyObBWyA7M2V_BWJGmO7W_j1e5TQF8PiD8ZEgNhnMd7jcEM0tGAH-v0aiV-X37gFq0bRkU3cb3xZo_4s_eLnTxfe270webbMo4-mIOFzClG7asdkgGQvNnO-aN9bvxBtii6aJVnYT5UIQDEofeJG-PdDVhkVyfSl__EHw`
const uri = "/auth/.well-known/openid-configuration/jwksX"

const jwks = `{
  "keys": [
    {
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "kid": "7f7355bfbf4f5f93df2bf879a78e223a",
      "e": "AQAB",
      "n": "q__qt-nUPcpLMVdfI8qSjQc-FHWVCd6UlH6XoAyVmPyfiwcYboDRTMyALTU_LHtEBb4EhxjxKOqQMHH6CZecFYeXnPgxx_PrxGRb4YVE95Aa3KHons1FATQqtd4Vjk29KvJXDxYjb_vWRUUnV8Ur-FYsYORH053ivq0lWbzYdMsOcXIrvjL-Zqb2ctVDKzqXEH_EE44iX9IXAHGVRCi7EvsSnVXT0CPivvsokIdQMPcT5TX4WGo0rfteDNNoRBEnu4Ra-9zoEAdb0knLdtgzIzIm5Bvg-ybkuqr85pV5Mgx6ojvzenW0Ongz00Mv-7hbbY0Kv8HIZb5smr-zvmCW3w"
    }
  ]
}`

var noTLS = tls.Config{}
