package service_test

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"

	"github.com/plgd-dev/cloud/oauth-server/service"
	"github.com/plgd-dev/cloud/oauth-server/test"
	"github.com/plgd-dev/cloud/oauth-server/uri"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/plgd-dev/kit/codec/json"
	"github.com/plgd-dev/kit/security/jwt"
	"github.com/stretchr/testify/require"
)

func TestRequestHandler_getUItoken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, service.ClientUI, "https://localhost:3000", http.StatusTemporaryRedirect)
	token := getToken(t, service.ClientUI, code, service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)

	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	_, err := validator.Parse(token["access_token"])
	require.NoError(t, err)
	_, err = validator.Parse(token["id_token"])
	require.NoError(t, err)
}

func TestRequestHandler_getServiceToken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	token := getToken(t, service.ClientService, "", service.AllowedGrantType_CLIENT_CREDENTIALS, http.StatusOK)

	validator := jwt.NewValidator(fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, uri.JWKs), &tls.Config{
		InsecureSkipVerify: true,
	})
	_, err := validator.Parse(token["access_token"])
	require.NoError(t, err)
}

func TestRequestHandler_getDeviceToken(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	code := getAuthorize(t, service.ClientDevice, "https://localhost:3000", http.StatusTemporaryRedirect)
	token := getToken(t, service.ClientDevice, code, service.AllowedGrantType_AUTHORIZATION_CODE, http.StatusOK)

	require.Equal(t, service.ClientDevice, token["access_token"])
}

func getToken(t *testing.T, clientID, code string, grantType service.AllowedGrantType, statusCode int) map[string]string {
	reqBody := map[string]string{
		"grant_type":         string(grantType),
		uri.ClientIDQueryKey: clientID,
		"code":               code,
	}
	d, err := json.Encode(reqBody)
	require.NoError(t, err)

	getReq := test.NewRequest(http.MethodPost, uri.Token, bytes.NewReader(d)).Build()
	res := test.HTTPDo(t, getReq, false)
	defer res.Body.Close()
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
