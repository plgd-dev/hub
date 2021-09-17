package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValid(t *testing.T) {
	c := testClaims()
	require.NoError(t, c.ValidTimes(time.Now()))
}

func TestExpired(t *testing.T) {
	c := testClaims()
	c[ClaimExpiresAt] = now.Add(-time.Hour).Unix()
	require.Error(t, c.ValidTimes(time.Now()))
}

func TestIssuedLater(t *testing.T) {
	c := testClaims()
	c[ClaimIssuedAt] = now.Add(time.Hour).Unix()
	require.Error(t, c.ValidTimes(time.Now()))
}

func TestNotBefore(t *testing.T) {
	c := testClaims()
	c[ClaimNotBefore] = now.Add(time.Hour).Unix()
	require.Error(t, c.ValidTimes(time.Now()))
}

func TestEmptyAudience(t *testing.T) {
	c := testClaims()
	require.Nil(t, c.Audience())
}

func TestAudienceOfOne(t *testing.T) {
	aud := "test"
	c := testClaims()
	c[ClaimAudience] = aud
	require.Equal(t, []string{aud}, c.Audience())
}

func TestAudienceOfTwo(t *testing.T) {
	c := testClaims()
	c[ClaimAudience] = []interface{}{"test1", "test2"}
	require.Equal(t, []string{"test1", "test2"}, c.Audience())
}

var now = time.Now()

func testClaims() Claims {
	return Claims{
		ClaimClientID:  "testClientID",
		ClaimEmail:     "testEmail",
		ClaimScope:     []string{"testScope"},
		ClaimExpiresAt: now.Add(time.Hour).Unix(),
		ClaimIssuedAt:  now.Unix(),
		ClaimNotBefore: now.Unix(),
	}
}
