package service_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/service"
	m2mOauthServerTest "github.com/plgd-dev/hub/v2/m2m-oauth-server/test"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/require"
)

func TestGetToken(t *testing.T) {
	const deviceID = "deviceId"
	const owner = "owner"
	type want struct {
		deviceID interface{}
		owner    interface{}
	}

	oauthServerTeardown := test.SetUp(t)
	defer oauthServerTeardown()

	token := test.GetDefaultAccessToken(t)

	invalidToken := config.CreateJwtToken(t, jwt.MapClaims{
		"sub": "aaa",
		"iss": "https://invalid-issuer",
	})

	tests := []struct {
		name     string
		args     m2mOauthServerTest.AccessTokenOptions
		wantCode int
		want     want
	}{
		{
			name: "serviceToken",
			args: m2mOauthServerTest.AccessTokenOptions{
				ClientID:     m2mOauthServerTest.ServiceOAuthClient.ID,
				ClientSecret: m2mOauthServerTest.GetSecret(t, m2mOauthServerTest.ServiceOAuthClient.ID),
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
			},
			wantCode: http.StatusOK,
		},
		{
			name: "snippetServiceToken",
			args: m2mOauthServerTest.AccessTokenOptions{
				ClientID:     m2mOauthServerTest.JWTPrivateKeyOAuthClient.ID,
				ClientSecret: m2mOauthServerTest.GetSecret(t, m2mOauthServerTest.JWTPrivateKeyOAuthClient.ID),
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
				Owner:        owner,
				PostForm:     true,
			},
			wantCode: http.StatusOK,
			want: want{
				owner: owner,
			},
		},
		{
			name: "snippetServiceToken - JWT",
			args: m2mOauthServerTest.AccessTokenOptions{
				ClientID:  m2mOauthServerTest.JWTPrivateKeyOAuthClient.ID,
				GrantType: string(service.GrantTypeClientCredentials),
				Host:      config.M2M_OAUTH_SERVER_HTTP_HOST,
				JWT:       token,
			},
			wantCode: http.StatusOK,
			want: want{
				owner: "1",
			},
		},
		{
			name: "snippetServiceToken - invalid JWT",
			args: m2mOauthServerTest.AccessTokenOptions{
				ClientID:  m2mOauthServerTest.JWTPrivateKeyOAuthClient.ID,
				GrantType: string(service.GrantTypeClientCredentials),
				Host:      config.M2M_OAUTH_SERVER_HTTP_HOST,
				JWT:       invalidToken,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "snippetServiceToken - invalid owner",
			args: m2mOauthServerTest.AccessTokenOptions{
				ClientID:     m2mOauthServerTest.JWTPrivateKeyOAuthClient.ID,
				ClientSecret: m2mOauthServerTest.GetSecret(t, m2mOauthServerTest.JWTPrivateKeyOAuthClient.ID),
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "deviceProvisioningServiceToken",
			args: m2mOauthServerTest.AccessTokenOptions{
				ClientID:     m2mOauthServerTest.DeviceProvisioningServiceOAuthClient.ID,
				ClientSecret: m2mOauthServerTest.GetSecret(t, m2mOauthServerTest.DeviceProvisioningServiceOAuthClient.ID),
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
				DeviceID:     deviceID,
				Owner:        owner,
			},
			wantCode: http.StatusOK,
			want: want{
				owner:    owner,
				deviceID: deviceID,
			},
		},
		{
			name: "deviceProvisioningServiceToken - invalid owner",
			args: m2mOauthServerTest.AccessTokenOptions{
				ClientID:     m2mOauthServerTest.DeviceProvisioningServiceOAuthClient.ID,
				ClientSecret: m2mOauthServerTest.GetSecret(t, m2mOauthServerTest.DeviceProvisioningServiceOAuthClient.ID),
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
				DeviceID:     deviceID,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "deviceProvisioningServiceToken - invalid deviceID",
			args: m2mOauthServerTest.AccessTokenOptions{
				ClientID:     m2mOauthServerTest.DeviceProvisioningServiceOAuthClient.ID,
				ClientSecret: m2mOauthServerTest.GetSecret(t, m2mOauthServerTest.DeviceProvisioningServiceOAuthClient.ID),
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
				Owner:        owner,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "deviceProvisioningServiceToken - invalid client",
			args: m2mOauthServerTest.AccessTokenOptions{
				ClientID:     m2mOauthServerTest.DeviceProvisioningServiceOAuthClient.ID,
				ClientSecret: m2mOauthServerTest.GetSecret(t, m2mOauthServerTest.DeviceProvisioningServiceOAuthClient.ID),
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
				Owner:        owner,
			},
			wantCode: http.StatusBadRequest,
		},
	}

	cfg := m2mOauthServerTest.MakeConfig(t)
	fmt.Printf("cfg: %v\n", cfg)

	webTearDown := m2mOauthServerTest.SetUp(t)
	defer webTearDown()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := m2mOauthServerTest.GetAccessToken(t, tt.wantCode, m2mOauthServerTest.WithAccessTokenOptions(tt.args))
			if tt.wantCode != http.StatusOK {
				return
			}
			validator := m2mOauthServerTest.GetJWTValidator(fmt.Sprintf("https://%s%s", config.M2M_OAUTH_SERVER_HTTP_HOST, uri.JWKs))
			claims, err := validator.Parse(token[uri.AccessTokenKey])
			require.NoError(t, err)
			require.Equal(t, tt.want.deviceID, claims[m2mOauthServerTest.DeviceIDClaim])
			require.Equal(t, tt.want.owner, claims[m2mOauthServerTest.OwnerClaim])
		})
	}
}
