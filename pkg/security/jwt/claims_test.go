package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValid(t *testing.T) {
	c := testClaims()
	require.NoError(t, c.Valid())
}

func TestExpired(t *testing.T) {
	c := testClaims()
	c.StandardClaims.ExpiresAt = now.Add(-time.Hour).Unix()
	require.Error(t, c.Valid())
}

func TestIssuedLater(t *testing.T) {
	c := testClaims()
	c.StandardClaims.IssuedAt = now.Add(time.Hour).Unix()
	require.Error(t, c.Valid())
}

func TestNotBefore(t *testing.T) {
	c := testClaims()
	c.StandardClaims.NotBefore = now.Add(time.Hour).Unix()
	require.Error(t, c.Valid())
}

func TestEmptyAudience(t *testing.T) {
	c := testClaims()
	require.Nil(t, c.GetAudience())
}

func TestAudienceOfOne(t *testing.T) {
	aud := "test"
	c := testClaims()
	c.Audience = aud
	require.Equal(t, []string{aud}, c.GetAudience())
}

func TestAudienceOfTwo(t *testing.T) {
	c := testClaims()
	c.Audience = []interface{}{"test1", "test2"}
	require.Equal(t, []string{"test1", "test2"}, c.GetAudience())
}

var now = time.Now()

func testClaims() Claims {
	return Claims{
		ClientID: "testClientID",
		Email:    "testEmail",
		Scope:    []string{"testScope"},
		StandardClaims: StandardClaims{
			ExpiresAt: now.Add(time.Hour).Unix(),
			IssuedAt:  now.Unix(),
			NotBefore: now.Unix(),
		},
	}
}
