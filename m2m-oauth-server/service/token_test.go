package service_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/v2/m2m-oauth-server/service"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/test"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestGetToken(t *testing.T) {
	const deviceID = "deviceId"
	const owner = "owner"
	type want struct {
		deviceID interface{}
		owner    interface{}
	}

	tests := []struct {
		name     string
		args     test.AccessTokenOptions
		wantCode int
		want     want
	}{
		{
			name: "serviceToken",
			args: test.AccessTokenOptions{
				ClientID:     test.ServiceOAuthClient.ID,
				ClientSecret: test.ServiceOAuthClient.ClientSecret,
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
			},
			wantCode: http.StatusOK,
		},
		{
			name: "snippetServiceToken",
			args: test.AccessTokenOptions{
				ClientID:     test.SnippetServiceOAuthClient.ID,
				ClientSecret: test.SnippetServiceOAuthClient.ClientSecret,
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
				Owner:        owner,
			},
			wantCode: http.StatusOK,
			want: want{
				owner: owner,
			},
		},
		{
			name: "snippetServiceToken - invalid owner",
			args: test.AccessTokenOptions{
				ClientID:     test.SnippetServiceOAuthClient.ID,
				ClientSecret: test.SnippetServiceOAuthClient.ClientSecret,
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "deviceProvisioningServiceToken",
			args: test.AccessTokenOptions{
				ClientID:     test.DeviceProvisioningServiceOAuthClient.ID,
				ClientSecret: test.DeviceProvisioningServiceOAuthClient.ClientSecret,
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
			args: test.AccessTokenOptions{
				ClientID:     test.DeviceProvisioningServiceOAuthClient.ID,
				ClientSecret: test.DeviceProvisioningServiceOAuthClient.ClientSecret,
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
				DeviceID:     deviceID,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "deviceProvisioningServiceToken - invalid deviceID",
			args: test.AccessTokenOptions{
				ClientID:     test.DeviceProvisioningServiceOAuthClient.ID,
				ClientSecret: test.DeviceProvisioningServiceOAuthClient.ClientSecret,
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
				Owner:        owner,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "deviceProvisioningServiceToken - invalid client",
			args: test.AccessTokenOptions{
				ClientID:     test.DeviceProvisioningServiceOAuthClient.ID,
				ClientSecret: test.DeviceProvisioningServiceOAuthClient.ClientSecret,
				GrantType:    string(service.GrantTypeClientCredentials),
				Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
				Owner:        owner,
			},
			wantCode: http.StatusBadRequest,
		},
	}

	cfg := test.MakeConfig(t)
	fmt.Printf("cfg: %v\n", cfg)

	webTearDown := test.SetUp(t)
	defer webTearDown()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := test.GetAccessToken(t, tt.wantCode, test.WithAccessTokenOptions(tt.args))
			if tt.wantCode != http.StatusOK {
				return
			}
			validator := test.GetJWTValidator(fmt.Sprintf("https://%s%s", config.M2M_OAUTH_SERVER_HTTP_HOST, uri.JWKs))
			claims, err := validator.Parse(token[uri.AccessTokenKey])
			require.NoError(t, err)
			require.Equal(t, tt.want.deviceID, claims[test.DeviceIDClaim])
			require.Equal(t, tt.want.owner, claims[test.OwnerClaim])
		})
	}
}
