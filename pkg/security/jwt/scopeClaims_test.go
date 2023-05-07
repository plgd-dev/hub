package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testScopeClaims(scope ...string) *ScopeClaims {
	c := NewScopeClaims(scope...)
	c.Claims[ClaimClientID] = "testClientID"
	c.Claims[ClaimEmail] = "testEmail"
	c.Claims[ClaimScope] = []string{"testScope", "otherScope"}
	c.Claims[ClaimExpirationTime] = float64(now.Add(time.Hour).Unix())
	c.Claims[ClaimIssuedAt] = float64(now.Unix())
	c.Claims[ClaimNotBefore] = float64(now.Unix())
	return c
}

func TestScopeClaimsValidScope(t *testing.T) {
	c := testScopeClaims("testScope")
	require.NoError(t, c.Validate())
	c1 := testScopeClaims()
	require.NoError(t, c1.Validate())
}

func TestScopeClaimsMissingPredefinedScope(t *testing.T) {
	c := ScopeClaims{}
	err := c.Validate()
	require.ErrorIs(t, err, ErrMissingRequiredScopes)
}

func TestScopeClaimsMissingScope(t *testing.T) {
	c := testScopeClaims("invalid")
	err := c.Validate()
	require.ErrorIs(t, err, ErrMissingRequiredScopes)
}

func TestScopeClaimsExpiredScope(t *testing.T) {
	c := testScopeClaims("testScope")
	c.Claims[ClaimExpirationTime] = float64(now.Add(-time.Hour).Unix())
	err := c.Validate()
	require.ErrorIs(t, err, ErrTokenExpired)
}

func TestScopeClaimsNotYetIssued(t *testing.T) {
	c := testScopeClaims("testScope")
	c.Claims[ClaimIssuedAt] = float64(now.Add(time.Hour).Unix())
	err := c.Validate()
	require.ErrorIs(t, err, ErrTokenNotYetIssued)
}

func TestScopeClaimsNotYetValid(t *testing.T) {
	c := testScopeClaims("testScope")
	c.Claims[ClaimNotBefore] = float64(now.Add(time.Hour).Unix())
	err := c.Validate()
	require.ErrorIs(t, err, ErrTokenNotYetValid)
}
