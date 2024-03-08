package jwt_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func testScopeClaims(scope ...string) *pkgJwt.ScopeClaims {
	c := pkgJwt.NewScopeClaims(scope...)
	(*c)[pkgJwt.ClaimClientID] = "testClientID"
	(*c)[pkgJwt.ClaimEmail] = "testEmail"
	(*c)[pkgJwt.ClaimScope] = []string{"testScope", "otherScope"}
	(*c)[pkgJwt.ClaimExpirationTime] = float64(now.Add(time.Hour).Unix())
	(*c)[pkgJwt.ClaimIssuedAt] = float64(now.Unix())
	(*c)[pkgJwt.ClaimNotBefore] = float64(now.Unix())
	return c
}

func TestScopeClaimsGet(t *testing.T) {
	c1 := testScopeClaims()
	_, err := c1.GetExpirationTime()
	require.NoError(t, err)
	_, err = c1.GetIssuedAt()
	require.NoError(t, err)
	_, err = c1.GetNotBefore()
	require.NoError(t, err)
	_, err = c1.GetIssuer()
	require.NoError(t, err)
	_, err = c1.GetSubject()
	require.NoError(t, err)
	_, err = c1.GetAudience()
	require.NoError(t, err)
}

func TestScopeClaimsValidScope(t *testing.T) {
	c := testScopeClaims("testScope")
	require.NoError(t, c.Validate())
	c1 := testScopeClaims()
	require.NoError(t, c1.Validate())

	c2 := testScopeClaims()
	(*c2)[pkgJwt.PlgdRequiredScope] = nil
	require.NoError(t, c2.Validate())
}

func TestScopeClaimsInvalidScope(t *testing.T) {
	// invalid ClaimScope type
	c1 := testScopeClaims("requiredScope")
	(*c1)[pkgJwt.ClaimScope] = 42
	require.Error(t, c1.Validate())

	c2 := testScopeClaims("requiredScope")
	(*c2)[pkgJwt.ClaimScope] = "testScope"
	require.ErrorIs(t, c2.Validate(), pkgJwt.ErrMissingRequiredScopes)
}

func checkScopedClaims(t *testing.T, tokenClaims jwt.Claims, expError error) {
	token := config.CreateJwtToken(t, tokenClaims)
	_, err := jwt.ParseWithClaims(token, &pkgJwt.ScopeClaims{}, func(*jwt.Token) (interface{}, error) {
		return jwt.VerificationKeySet{
			Keys: []jwt.VerificationKey{
				[]uint8(config.JWTSecret),
			},
		}, nil
	}, jwt.WithIssuedAt())
	if expError == nil {
		require.NoError(t, err)
		return
	}
	require.ErrorIs(t, err, expError)
}

func TestScopeClaimsMissingPredefinedScope(t *testing.T) {
	c := pkgJwt.ScopeClaims{}
	checkScopedClaims(t, &c, pkgJwt.ErrMissingRequiredScopes)
}

func TestScopeClaimsMissingScope(t *testing.T) {
	c := testScopeClaims("invalid")
	checkScopedClaims(t, c, pkgJwt.ErrMissingRequiredScopes)
}

func TestScopeClaimsExpiredScope(t *testing.T) {
	c := testScopeClaims("testScope")
	(*c)[pkgJwt.ClaimExpirationTime] = float64(now.Add(-time.Hour).Unix())
	checkScopedClaims(t, c, jwt.ErrTokenExpired)
}

func TestScopeClaimsNotYetIssued(t *testing.T) {
	c := testScopeClaims("testScope")
	(*c)[pkgJwt.ClaimIssuedAt] = float64(now.Add(time.Hour).Unix())
	checkScopedClaims(t, c, jwt.ErrTokenUsedBeforeIssued)
}

func TestScopeClaimsNotYetValid(t *testing.T) {
	c := testScopeClaims("testScope")
	(*c)[pkgJwt.ClaimNotBefore] = float64(now.Add(time.Hour).Unix())
	checkScopedClaims(t, c, jwt.ErrTokenNotValidYet)
}
