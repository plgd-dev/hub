package jwt_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator(t *testing.T) {
	server := newTestJwks()
	defer server.Close()

	v := test.GetJWTValidator(server.URL + uri)
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

	v := test.GetJWTValidator(server.URL + uri)
	var c pkgJwt.Claims
	err := v.ParseWithClaims(token, &c)
	require.ErrorIs(t, err, pkgJwt.ErrTokenExpired)

	clientID, err := c.GetClientID()
	assert.NoError(t, err)
	assert.Equal(t, "test.client.id", clientID)
	email, err := c.GetEmail()
	assert.NoError(t, err)
	assert.Equal(t, "user@example.com", email)
	scope, err := c.GetScope()
	assert.Contains(t, scope, "test.scope")
	assert.NoError(t, err)
	audience, err := c.GetAudience()
	assert.NoError(t, err)
	assert.Contains(t, audience, "http://identity-server:3001/resources")
	assert.Contains(t, audience, "test.resource")
	exp, err := c.GetExpirationTime()
	assert.NoError(t, err)
	assert.Equal(t, 2019, exp.Year())
	id, err := c.GetID()
	assert.NoError(t, err)
	assert.Empty(t, id)
	iat, err := c.GetIssuedAt()
	assert.NoError(t, err)
	assert.Nil(t, iat)
	iss, err := c.GetIssuer()
	assert.NoError(t, err)
	assert.Equal(t, iss, "http://identity-server:3001")
	nbf, err := c.GetNotBefore()
	assert.NoError(t, err)
	assert.Equal(t, 2019, nbf.Year())
	sub, err := c.GetSubject()
	assert.NoError(t, err)
	assert.Equal(t, "1b87effa-34e2-4a44-82c6-6e0ab80209ff", sub)
}

func TestParser(t *testing.T) {
	server := newTestJwks()
	defer server.Close()

	v := test.GetJWTValidator(server.URL + uri)
	c, err := v.Parse(token)
	require.ErrorIs(t, err, jwt.ErrTokenExpired)

	assert.Equal(t, "test.client.id", c[pkgJwt.ClaimClientID])
	assert.Equal(t, "user@example.com", c[pkgJwt.ClaimEmail])
	assert.Contains(t, c[pkgJwt.ClaimScope], "test.scope")

	assert.Equal(t, "local", c["idp"])
	assert.Contains(t, c["amr"], "pwd")
	assert.Contains(t, c[pkgJwt.ClaimAudience], "http://identity-server:3001/resources")
	assert.Contains(t, c[pkgJwt.ClaimAudience], "test.resource")
	assert.Equal(t, "http://identity-server:3001", c[pkgJwt.ClaimIssuer])
	assert.Equal(t, "1b87effa-34e2-4a44-82c6-6e0ab80209ff", c[pkgJwt.ClaimSubject])

	assert.Equal(t, 2019, time.Unix(int64(c[pkgJwt.ClaimNotBefore].(float64)), 0).Year())
	assert.Equal(t, 2019, time.Unix(int64(c[pkgJwt.ClaimExpirationTime].(float64)), 0).Year())
	assert.Equal(t, 2019, time.Unix(int64(c["auth_time"].(float64)), 0).Year())
}

func TestEmptyToken(t *testing.T) {
	server := newTestJwks()
	defer server.Close()

	v := test.GetJWTValidator(server.URL + uri)
	_, err := v.Parse("")
	require.ErrorIs(t, err, pkgJwt.ErrMissingToken)

	var c pkgJwt.Claims
	err = v.ParseWithClaims("", &c)
	require.ErrorIs(t, err, pkgJwt.ErrMissingToken)
}

func TestInvalidToken(t *testing.T) {
	server := newTestJwks()
	defer server.Close()

	v := test.GetJWTValidator(server.URL + uri)
	_, err := v.Parse("invalid")
	require.ErrorIs(t, err, pkgJwt.ErrCannotParseToken)

	var c pkgJwt.Claims
	err = v.ParseWithClaims("invalid", &c)
	require.ErrorIs(t, err, pkgJwt.ErrCannotParseToken)
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
	jwt.MapClaims
	ClientID string   `json:"client_id"`
	Email    string   `json:"email"`
	Scope    []string `json:"scope"`
}

func (c testClaims) Validate() error {
	return nil
}

const (
	token = `eyJhbGciOiJSUzI1NiIsImtpZCI6IjdmNzM1NWJmYmY0ZjVmOTNkZjJiZjg3OWE3OGUyMjNhIiwidHlwIjoiSldUIn0.eyJuYmYiOjE1NjM4ODk5NTgsImV4cCI6MTU2Mzg5MzU1OCwiaXNzIjoiaHR0cDovL2lkZW50aXR5LXNlcnZlcjozMDAxIiwiYXVkIjpbImh0dHA6Ly9pZGVudGl0eS1zZXJ2ZXI6MzAwMS9yZXNvdXJjZXMiLCJ0ZXN0LnJlc291cmNlIl0sImNsaWVudF9pZCI6InRlc3QuY2xpZW50LmlkIiwic3ViIjoiMWI4N2VmZmEtMzRlMi00YTQ0LTgyYzYtNmUwYWI4MDIwOWZmIiwiYXV0aF90aW1lIjoxNTYzODg5ODQ1LCJpZHAiOiJsb2NhbCIsImVtYWlsIjoidXNlckBleGFtcGxlLmNvbSIsInRlbmFudF9pZCI6IjdlYWE1MzJhLWMyYmYtNDRiNC05MTA5LWE3ODBmNGY2MzI2NCIsImRlZmF1bHRfcHJvZHVjdF91cmkiOiIvIiwicm9sZXMiOlsidGVzdC5yb2xlMSIsInRlc3Qucm9sZTIiXSwic2NvcGUiOlsidGVzdC5zY29wZSJdLCJhbXIiOlsicHdkIl19.jwgEJpn9aYZrWFzMRvW9ABZpA_MnZDNZcfWJtFm-luyYBm2D06P6bsKTH0mYcxKTVSVHnEczRecQumwtWjyvyVVtCLDOW1GVPseChw-By152QxOIeOpsJh2zPEKlVaqvPkFKOfFGPWYb99RN8-Cfv9hKYtUCrFhHCwcWcrzCngiYElGvRkyObBWyA7M2V_BWJGmO7W_j1e5TQF8PiD8ZEgNhnMd7jcEM0tGAH-v0aiV-X37gFq0bRkU3cb3xZo_4s_eLnTxfe270webbMo4-mIOFzClG7asdkgGQvNnO-aN9bvxBtii6aJVnYT5UIQDEofeJG-PdDVhkVyfSl__EHw`
	uri   = "/auth/.well-known/openid-configuration/jwksX"
)

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
