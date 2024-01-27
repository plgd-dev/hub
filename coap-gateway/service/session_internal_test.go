//go:build test
// +build test

package service

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestAuthorizationContext(t *testing.T) {
	var acNil *authorizationContext
	require.Equal(t, "", acNil.GetDeviceID())
	require.Equal(t, "", acNil.GetAccessToken())
	require.Equal(t, "", acNil.GetUserID())
	require.Len(t, acNil.GetJWTClaims(), 0)

	const deviceID = "devID"
	token := config.CreateJwtToken(t, jwt.MapClaims{
		"owner": "owner",
	})
	const userID = "userID"
	ac := authorizationContext{
		DeviceID:    deviceID,
		AccessToken: token,
		UserID:      userID,
	}
	require.Equal(t, deviceID, ac.GetDeviceID())
	require.Equal(t, token, ac.GetAccessToken())
	require.Equal(t, userID, ac.GetUserID())
	require.Len(t, ac.GetJWTClaims(), 1)
	require.Equal(t, "owner", ac.GetJWTClaims()["owner"])

	acNoToken := authorizationContext{}
	require.Len(t, acNoToken.GetJWTClaims(), 0)
}

func TestAuthorizationContextIsValid(t *testing.T) {
	var acNil *authorizationContext
	require.Error(t, acNil.IsValid())

	acNoToken := authorizationContext{}
	require.Error(t, acNoToken.IsValid())

	acExpired := authorizationContext{
		Expire:      time.Now().Add(-time.Hour),
		AccessToken: config.CreateJwtToken(t, jwt.MapClaims{}),
	}
	require.Error(t, acExpired.IsValid())

	// valid - without expiration
	acNoExpire := authorizationContext{
		AccessToken: config.CreateJwtToken(t, jwt.MapClaims{}),
	}
	require.NoError(t, acNoExpire.IsValid())

	// valid - not expired
	acNotExpired := authorizationContext{
		Expire:      time.Now().Add(time.Hour),
		AccessToken: config.CreateJwtToken(t, jwt.MapClaims{}),
	}
	require.NoError(t, acNotExpired.IsValid())
}
