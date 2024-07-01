package service_test

import (
	"net/http"
	"testing"

	m2mOauthServerTest "github.com/plgd-dev/hub/v2/m2m-oauth-server/test"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

func TestRequestHandlerGetJWKs(t *testing.T) {
	oauthServerTeardown := test.SetUp(t)
	defer oauthServerTeardown()

	webTearDown := m2mOauthServerTest.SetUp(t)
	defer webTearDown()

	getJWKs(t)
}

func getJWKs(t *testing.T) map[string]interface{} {
	getReq := m2mOauthServerTest.NewRequestBuilder(http.MethodGet, config.M2M_OAUTH_SERVER_HTTP_HOST, uri.JWKs, nil).Build()
	res := m2mOauthServerTest.HTTPDo(t, getReq, false)
	defer func() {
		_ = res.Body.Close()
	}()

	var body map[string]interface{}
	err := json.ReadFrom(res.Body, &body)
	require.NoError(t, err)
	require.NotEmpty(t, body["keys"])
	require.Len(t, body["keys"].([]interface{}), 1)
	return body
}
