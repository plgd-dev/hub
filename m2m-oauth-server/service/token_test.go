package service_test

import (
	"context"
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
	type want struct {
		owner                    interface{}
		existOriginalTokenClaims bool
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
			name: "serviceToken - json",
			args: m2mOauthServerTest.AccessTokenOptions{
				Ctx:          context.Background(),
				ClientID:     m2mOauthServerTest.ServiceOAuthClient.ID,
				ClientSecret: m2mOauthServerTest.GetSecret(t, m2mOauthServerTest.ServiceOAuthClient.ID),
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
			},
			wantCode: http.StatusOK,
		},
		{
			name: "serviceToken - postForm",
			args: m2mOauthServerTest.AccessTokenOptions{
				Ctx:          context.Background(),
				ClientID:     m2mOauthServerTest.ServiceOAuthClient.ID,
				ClientSecret: m2mOauthServerTest.GetSecret(t, m2mOauthServerTest.ServiceOAuthClient.ID),
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
				PostForm:     true,
			},
			wantCode: http.StatusOK,
		},
		{
			name: "snippetServiceToken - JWT",
			args: m2mOauthServerTest.AccessTokenOptions{
				Ctx:       context.Background(),
				ClientID:  m2mOauthServerTest.JWTPrivateKeyOAuthClient.ID,
				GrantType: string(service.GrantTypeClientCredentials),
				Host:      config.M2M_OAUTH_SERVER_HTTP_HOST,
				JWT:       token,
			},
			wantCode: http.StatusOK,
			want: want{
				owner:                    "1",
				existOriginalTokenClaims: true,
			},
		},
		{
			name: "invalid client",
			args: m2mOauthServerTest.AccessTokenOptions{
				Ctx:       context.Background(),
				ClientID:  "invalid client",
				GrantType: string(service.GrantTypeClientCredentials),
				Host:      config.M2M_OAUTH_SERVER_HTTP_HOST,
				JWT:       invalidToken,
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "snippetServiceToken - invalid JWT",
			args: m2mOauthServerTest.AccessTokenOptions{
				Ctx:       context.Background(),
				ClientID:  m2mOauthServerTest.JWTPrivateKeyOAuthClient.ID,
				GrantType: string(service.GrantTypeClientCredentials),
				Host:      config.M2M_OAUTH_SERVER_HTTP_HOST,
				JWT:       invalidToken,
			},
			wantCode: http.StatusUnauthorized,
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
			require.Equal(t, tt.want.owner, claims[m2mOauthServerTest.OwnerClaim])
			if tt.want.existOriginalTokenClaims {
				require.NotEmpty(t, claims[uri.OriginalTokenClaims])
			}
		})
	}
}
