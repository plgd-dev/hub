package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidScope(t *testing.T) {
	c := testScopeClaims("testScope")
	require.NoError(t, c.Valid())
	c1 := testScopeClaims()
	require.NoError(t, c1.Valid())
}

func TestInvalidScope(t *testing.T) {
	c := testScopeClaims("invalid")
	err := c.Valid()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing scopes")
}

func TestExpiredScope(t *testing.T) {
	c := testScopeClaims("testScope")
	c.Claims[ClaimExpiresAt] = now.Add(-time.Hour).Unix()
	err := c.Valid()
	require.Error(t, err)
	require.Contains(t, err.Error(), "token is expired")
}

func testScopeClaims(scope ...string) *ScopeClaims {
	c := NewScopeClaims(scope...)
	c.Claims = Claims{
		ClaimClientID:  "testClientID",
		ClaimEmail:     "testEmail",
		ClaimScope:     []string{"testScope", "otherScope"},
		ClaimExpiresAt: now.Add(time.Hour).Unix(),
		ClaimIssuedAt:  now.Unix(),
		ClaimNotBefore: now.Unix(),
	}
	return c
}
