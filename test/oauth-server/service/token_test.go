package service_test

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/pkg/security/jwt"
	"github.com/plgd-dev/hub/test/config"
	"github.com/plgd-dev/hub/test/oauth-server/service"
	"github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/hub/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

func TestRequestHandler_getUItoken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTest, "https://localhost:3000", "nonse", "", "", "", http.StatusFound, false, false)
	token := getToken(t, test.ClientTest, "", "localhost", "", code, "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)

	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	accessToken, err := validator.Parse(token["access_token"])
	require.NoError(t, err)
	require.Empty(t, accessToken[service.TokenDeviceID])
	_, err = validator.Parse(token["id_token"])
	require.NoError(t, err)
}

func TestRequestHandler_getServiceToken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	token := getToken(t, test.ClientTest, "", "localhost", "", "", "", service.AllowedGrantType_CLIENT_CREDENTIALS, http.StatusOK)

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
	token := getToken(t, test.ClientTest, "", "", "", code, "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)

	require.NotEmpty(t, token["access_token"])
	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	accessToken, err := validator.Parse(token["access_token"])
	require.NoError(t, err)
	require.NotEmpty(t, accessToken[service.TokenDeviceID])
}

func TestRequestHandlerGetTokenWithDefaultScopes(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTest, "", "https://localhost:3000", "", "", "", http.StatusFound, false, false)
	token := getToken(t, test.ClientTest, "", "", "", code, "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)

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
	token := getToken(t, test.ClientTest, "", "", "", code, "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)

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
	getToken(t, test.ClientTestRequiredParams, "blabla", "", "http://localhost:7777", code, "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusBadRequest)
}

func TestRequestHandlerGetTokenWithInvalidRedirectURI(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTestRequiredParams, "", "http://localhost:7777", "", "r:*", "code", http.StatusFound, false, false)
	getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "", "https://localhost:3232", code, "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusBadRequest)
}

func TestRequestHandlerGetTokenWithInvalidCode(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	getAuthorize(t, test.ClientTestRequiredParams, "", "http://localhost:7777", "", "r:*", "code", http.StatusFound, false, false)
	getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "", "http://localhost:7777", "123", "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusBadRequest)
}

func TestRequestHandlerGetTokenWithDuplicitExchange(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTestRequiredParams, "", "http://localhost:7777", "", "r:*", "code", http.StatusFound, false, false)
	getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "", "http://localhost:7777", code, "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)
	getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "", "http://localhost:7777", code, "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusBadRequest)
}

func TestRequestHandlerGetTokenWithValidRequiredParams(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, test.ClientTestRequiredParams, "", "http://localhost:7777", "", "r:*", "code", http.StatusFound, false, false)
	token := getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "", "http://localhost:7777", code, "", service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)
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

	getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "invalidRefreshToken", "http://localhost:7777", "", "refreshToken", service.AllowedGrantType_REFRESH_TOKEN, http.StatusOK)
	getToken(t, test.ClientTestRequiredParams, test.ClientTestRequiredParamsSecret, "invalidRefreshToken", "http://localhost:7777", "", "refreshToken", service.AllowedGrantType_REFRESH_TOKEN, http.StatusBadRequest)
}

func TestRequestHandlerGetTokenWithInvalidRefreshToken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	getToken(t, test.ClientTest, test.ClientTestRequiredParamsSecret, "invalidRefreshToken", "http://localhost:7777", "", "refreshokeninvalid", service.AllowedGrantType_REFRESH_TOKEN, http.StatusBadRequest)
}

func TestRequestHandlerGetTokenWithValidRefreshToken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	token := getToken(t, test.ClientTest, test.ClientTestRequiredParamsSecret, "invalidRefreshToken", "http://localhost:7777", "", "refresh-token", service.AllowedGrantType_REFRESH_TOKEN, http.StatusOK)
	require.NotEmpty(t, token["access_token"])
	require.Equal(t, token["scope"], service.DefaultScope)
	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	accessToken, err := validator.Parse(token["access_token"])
	require.NoError(t, err)
	require.Equal(t, service.DefaultScope, accessToken[service.TokenScopeKey])
}

func getToken(t *testing.T, clientID, clientSecret, audience, redirectURI, code, refreshToken string, grantType service.AllowedGrantType, statusCode int) map[string]string {
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
		code := body["access_token"]
		require.NotEmpty(t, code)
		return body
	}

	return nil
}
