package jwt_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testClaims struct {
	jwt.MapClaims
	ClientID string   `json:"client_id"`
	Email    string   `json:"email"`
	Scope    []string `json:"scope"`
}

func (c testClaims) Validate() error {
	return nil
}

func testToken(t *testing.T) string {
	return security.CreateJwksToken(t, jwt.MapClaims{
		pkgJwt.ClaimNotBefore:      1563889958,
		pkgJwt.ClaimExpirationTime: 1563893558,
		pkgJwt.ClaimIssuer:         "http://identity-server:3001",
		pkgJwt.ClaimAudience:       []string{"http://identity-server:3001/resources", "test.resource"},
		pkgJwt.ClaimClientID:       "test.client.id",
		pkgJwt.ClaimSubject:        "1b87effa-34e2-4a44-82c6-6e0ab80209ff",
		"auth_time":                1563889845,
		"idp":                      "local",
		pkgJwt.ClaimEmail:          "user@example.com",
		"tenant_id":                "7eaa532a-c2bf-44b4-9109-a780f4f63264",
		"default_product_uri":      "/",
		"roles":                    []string{"test.role1", "test.role2"},
		pkgJwt.ClaimScope:          []string{"test.scope"},
		"amr":                      []string{"pwd"},
	})
}

func TestValidator(t *testing.T) {
	jwks := security.NewTestJwks(t)
	defer jwks.Close()

	v := test.GetJWTValidator(jwks.URL())
	var c testClaims
	err := v.ParseWithClaims(context.Background(), testToken(t), &c)
	require.NoError(t, err)

	assert.Equal(t, "test.client.id", c.ClientID)
	assert.Equal(t, "user@example.com", c.Email)
	assert.Contains(t, c.Scope, "test.scope")
}

func TestClaims(t *testing.T) {
	jwks := security.NewTestJwks(t)
	defer jwks.Close()

	v := test.GetJWTValidator(jwks.URL())
	var c pkgJwt.Claims
	err := v.ParseWithClaims(context.Background(), testToken(t), &c)
	require.ErrorIs(t, err, jwt.ErrTokenExpired)

	clientID, err := c.GetClientID()
	require.NoError(t, err)
	require.Equal(t, "test.client.id", clientID)
	email, err := c.GetEmail()
	require.NoError(t, err)
	require.Equal(t, "user@example.com", email)
	scope, err := c.GetScope()
	require.Contains(t, scope, "test.scope")
	require.NoError(t, err)
	audience, err := c.GetAudience()
	require.NoError(t, err)
	require.Contains(t, audience, "http://identity-server:3001/resources")
	require.Contains(t, audience, "test.resource")
	exp, err := c.GetExpirationTime()
	require.NoError(t, err)
	require.Equal(t, 2019, exp.Year())
	id, err := c.GetID()
	require.NoError(t, err)
	require.Empty(t, id)
	iat, err := c.GetIssuedAt()
	require.NoError(t, err)
	require.Nil(t, iat)
	iss, err := c.GetIssuer()
	require.NoError(t, err)
	require.Equal(t, "http://identity-server:3001", iss)
	nbf, err := c.GetNotBefore()
	require.NoError(t, err)
	require.Equal(t, 2019, nbf.Year())
	sub, err := c.GetSubject()
	require.NoError(t, err)
	require.Equal(t, "1b87effa-34e2-4a44-82c6-6e0ab80209ff", sub)
}

func TestParser(t *testing.T) {
	jwks := security.NewTestJwks(t)
	defer jwks.Close()

	v := test.GetJWTValidator(jwks.URL())
	token := testToken(t)
	_, err := v.Parse(token)
	require.ErrorIs(t, err, jwt.ErrTokenExpired)

	c, err := pkgJwt.ParseToken(token)
	require.NoError(t, err)
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
	jwks := security.NewTestJwks(t)
	defer jwks.Close()

	v := test.GetJWTValidator(jwks.URL())
	_, err := v.Parse("")
	require.ErrorIs(t, err, pkgJwt.ErrMissingToken)

	var c pkgJwt.Claims
	err = v.ParseWithClaims(context.Background(), "", &c)
	require.ErrorIs(t, err, pkgJwt.ErrMissingToken)

	_, err = v.ParseWithContext(context.Background(), "")
	require.ErrorIs(t, err, pkgJwt.ErrMissingToken)
	_, err = v.ParseWithContext(context.Background(), "invalid")
	require.ErrorIs(t, err, pkgJwt.ErrCannotParseToken)
}

func TestInvalidToken(t *testing.T) {
	jwks := security.NewTestJwks(t)
	defer jwks.Close()

	v := test.GetJWTValidator(jwks.URL())
	_, err := v.Parse("invalid")
	require.ErrorIs(t, err, pkgJwt.ErrCannotParseToken)

	var c pkgJwt.Claims
	err = v.ParseWithClaims(context.Background(), "invalid", &c)
	require.ErrorIs(t, err, pkgJwt.ErrCannotParseToken)
}
