package service_test

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/oauth-server/service"
	"github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

func TestRequestHandler_getUItoken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTest, "https://localhost:3000", "nonse", "", "", "", http.StatusFound, false, false)
	token := getToken(t, test.ClientTest, "", "localhost", "", code, "", "", "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)

	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	accessToken, err := validator.Parse(token["access_token"])
	require.NoError(t, err)
	require.Empty(t, accessToken[uri.DeviceIDClaimKey])
	_, err = validator.Parse(token["id_token"])
	require.NoError(t, err)
}

func TestRequestHandler_getServiceToken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	token := getToken(t, test.ClientTest, "", "localhost", "", "", "", "", "", service.AllowedGrantType_CLIENT_CREDENTIALS, http.StatusOK)

	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	_, err := validator.Parse(token["access_token"])
	require.NoError(t, err)
}

func TestRequestHandler_getDeviceToken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTest, "", "https://localhost:3000", "abc", "", "", http.StatusFound, false, false)
	token := getToken(t, test.ClientTest, "", "", "", code, "", "", "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)

	require.NotEmpty(t, token["access_token"])
	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	accessToken, err := validator.Parse(token["access_token"])
	require.NoError(t, err)
	require.NotEmpty(t, accessToken[uri.DeviceIDClaimKey])
}

func TestRequestHandlerGetTokenWithDefaultScopes(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTest, "", "https://localhost:3000", "", "", "", http.StatusFound, false, false)
	token := getToken(t, test.ClientTest, "", "", "", code, "", "", "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)

	require.NotEmpty(t, token["access_token"])
	require.Equal(t, token["scope"], service.DefaultScope)
	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	accessToken, err := validator.Parse(token["access_token"])
	require.NoError(t, err)
	require.Equal(t, service.DefaultScope, accessToken[service.TokenScopeKey])
}

func TestRequestHandlerGetTokenWithCuscomScopes(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTest, "", "https://localhost:3000", "", "r:* w:*", "", http.StatusFound, false, false)
	token := getToken(t, test.ClientTest, "", "", "", code, "", "", "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)

	require.NotEmpty(t, token["access_token"])
	require.Equal(t, token["scope"], "r:* w:*")
	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	accessToken, err := validator.Parse(token["access_token"])
	require.NoError(t, err)
	require.Equal(t, "r:* w:*", accessToken[service.TokenScopeKey])
}

func TestRequestHandlerGetTokenWithInvalidSecret(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTestRequiredParams, "", "http://localhost:7777", "", "r:*", "code", http.StatusFound, false, false)
	getToken(t, test.ClientTestRequiredParams, "blabla", "", "http://localhost:7777", code, "", "", "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusBadRequest)
}

func TestRequestHandlerGetTokenWithInvalidRedirectURI(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTestRequiredParams, "", "http://localhost:7777", "", "r:*", "code", http.StatusFound, false, false)
	getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "", "https://localhost:3232", code, "", "", "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusBadRequest)
}

func TestRequestHandlerGetTokenWithInvalidCode(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	getAuthorize(t, test.ClientTestRequiredParams, "", "http://localhost:7777", "", "r:*", "code", http.StatusFound, false, false)
	getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "", "http://localhost:7777", "123", "", "", "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusBadRequest)
}

func TestRequestHandlerGetTokenWithDuplicitExchange(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTestRequiredParams, "", "http://localhost:7777", "", "r:*", "code", http.StatusFound, false, false)
	getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "", "http://localhost:7777", code, "", "", "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)
	getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "", "http://localhost:7777", code, "", "", "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusBadRequest)
}

func TestRequestHandlerGetTokenWithValidRequiredParams(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTestRequiredParams, "", "http://localhost:7777", "", "r:*", "code", http.StatusFound, false, false)
	token := getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "", "http://localhost:7777", code, "", "", "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)
	require.NotEmpty(t, token["access_token"])
	require.Equal(t, token["scope"], "r:*")
	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	accessToken, err := validator.Parse(token["access_token"])
	require.NoError(t, err)
	require.Equal(t, "r:*", accessToken[service.TokenScopeKey])
}

func TestRequestHandlerGetTokenWithDuplicitdRefreshToken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "invalidRefreshToken", "http://localhost:7777", "", "refreshToken", "", "", service.AllowedGrantType_REFRESH_TOKEN, http.StatusOK)
	getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "invalidRefreshToken", "http://localhost:7777", "", "refreshToken", "", "", service.AllowedGrantType_REFRESH_TOKEN, http.StatusBadRequest)
}

func TestRequestHandlerGetTokenWithInvalidRefreshToken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	getToken(t, test.ClientTest, test.ClientTestRequiredParamsSecret, "invalidRefreshToken", "http://localhost:7777", "", "refreshokeninvalid", "", "", service.AllowedGrantType_REFRESH_TOKEN, http.StatusBadRequest)
}

func TestRequestHandlerGetTokenWithValidRefreshToken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	token := getToken(t, test.ClientTest, test.ClientTestRequiredParamsSecret, "invalidRefreshToken", "http://localhost:7777", "", "refresh-token", "", "", service.AllowedGrantType_REFRESH_TOKEN, http.StatusOK)
	require.NotEmpty(t, token["access_token"])
	require.Equal(t, token["scope"], service.DefaultScope)
	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	accessToken, err := validator.Parse(token["access_token"])
	require.NoError(t, err)
	require.Equal(t, service.DefaultScope, accessToken[service.TokenScopeKey])
}

func TestGetRequestHandlerGetTokenWithValidClient(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	accessToken := getTokenUsingGet(t, test.ClientTest, false, http.StatusOK)
	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	_, err := validator.Parse(accessToken)
	require.NoError(t, err)
}

func TestGetRequestHandlerGetTokenWithDeviceIDAndOwnerClaim(t *testing.T) {
	type args struct {
		deviceID string
		owner    string
	}
	const deviceID = "deviceId"
	const owner = "owner"
	tests := []struct {
		name         string
		args         args
		wantDeviceID string
		wantOwner    string
	}{
		{
			name: deviceID,
			args: args{
				deviceID: deviceID,
			},
			wantDeviceID: deviceID,
		},
		{
			name: owner,
			args: args{
				owner: owner,
			},
			wantOwner: service.DeviceUserID,
		},
		{
			name: deviceID + "+" + owner,
			args: args{
				deviceID: deviceID,
				owner:    owner,
			},
			wantDeviceID: deviceID,
			wantOwner:    service.DeviceUserID,
		},
		{
			name: "empty",
		},
	}

	webTearDown := test.SetUp(t)
	defer webTearDown()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			token := getToken(t, test.ClientTest, "", "localhost", "", "", "", tt.args.deviceID, tt.args.owner, service.AllowedGrantType_CLIENT_CREDENTIALS, http.StatusOK)
			validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
				InsecureSkipVerify: true,
			})
			claims, err := validator.Parse(token["access_token"])
			require.NoError(t, err)
			if tt.wantDeviceID == "" {
				require.Empty(t, claims[uri.DeviceIDClaimKey])
			} else {
				require.Equal(t, claims[uri.DeviceIDClaimKey], tt.wantDeviceID)
			}
			if tt.wantOwner == "" {
				require.Empty(t, claims[uri.OwnerClaimKey])
			} else {
				// mock oauth server always set service.DeviceUserID, because it supports only one user
				require.Equal(t, claims[uri.OwnerClaimKey], service.DeviceUserID)
			}
		})
	}
}

func TestGetRequestHandlerGetTokenWithValidClientBasicAuth(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	accessToken := getTokenUsingGet(t, test.ClientTest, true, http.StatusOK)
	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	_, err := validator.Parse(accessToken)
	require.NoError(t, err)
}

func TestGetRequestHandlerGetTokenWithInvalidClient(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	getTokenUsingGet(t, "", false, http.StatusBadRequest)
}

func getTokenUsingGet(t *testing.T, clientID string, useBasicAuth bool, statusCode int) string {
	u, err := url.Parse(uri.Token)
	require.NoError(t, err)
	q, err := url.ParseQuery(u.RawQuery)
	require.NoError(t, err)
	q.Add(uri.ClientIDKey, clientID)

	u.RawQuery = q.Encode()
	getReq := test.NewRequest(http.MethodGet, config.OAUTH_SERVER_HOST, u.String(), nil).Build()
	if useBasicAuth {
		getReq.SetBasicAuth(clientID, "")
	}
	res := test.HTTPDo(t, getReq, false)
	defer func() {
		_ = res.Body.Close()
	}()
	require.Equal(t, statusCode, res.StatusCode)
	if res.StatusCode == http.StatusOK {
		require.Equal(t, http.StatusOK, res.StatusCode)
		var body map[string]string
		err = json.ReadFrom(res.Body, &body)
		require.NoError(t, err)
		accessToken := body["access_token"]
		require.NotEmpty(t, accessToken)
		return accessToken
	}
	return ""
}

func getToken(t *testing.T, clientID, clientSecret, audience, redirectURI, code, refreshToken, deviceID, owner string, grantType service.AllowedGrantType, statusCode int) map[string]string {
	reqBody := map[string]string{
		uri.GrantTypeKey:   string(grantType),
		uri.ClientIDKey:    clientID,
		uri.CodeKey:        code,
		uri.AudienceKey:    audience,
		uri.RedirectURIKey: redirectURI,
	}
	if refreshToken != "" {
		reqBody[uri.RefreshTokenKey] = refreshToken
	}
	if deviceID != "" {
		reqBody[uri.DeviceIDClaimKey] = deviceID
	}
	if owner != "" {
		reqBody[uri.OwnerClaimKey] = owner
	}

	d, err := json.Encode(reqBody)
	require.NoError(t, err)

	getReq := test.NewRequest(http.MethodPost, config.OAUTH_SERVER_HOST, uri.Token, bytes.NewReader(d)).Build()
	getReq.SetBasicAuth(clientID, clientSecret)
	res := test.HTTPDo(t, getReq, false)
	defer func() {
		_ = res.Body.Close()
	}()
	require.Equal(t, statusCode, res.StatusCode)
	if res.StatusCode == http.StatusOK {
		var body map[string]string
		err := json.ReadFrom(res.Body, &body)
		require.NoError(t, err)
		accessToken := body["access_token"]
		require.NotEmpty(t, accessToken)
		return body
	}
	return nil
}
