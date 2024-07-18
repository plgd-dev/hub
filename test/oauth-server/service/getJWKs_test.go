package service_test

import (
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

func TestRequestHandlerGetJWKs(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	getJWKs(t)
}

func getJWKs(t *testing.T) map[string]interface{} {
	getReq := test.NewRequestBuilder(http.MethodGet, config.OAUTH_SERVER_HOST, uri.JWKs, nil).Build()
	res := test.HTTPDo(t, getReq, false)
	defer func() {
		_ = res.Body.Close()
	}()

	var body map[string]interface{}
	err := json.ReadFrom(res.Body, &body)
	require.NoError(t, err)
	require.NotEmpty(t, body["keys"])
	require.Len(t, body["keys"].([]interface{}), 2)
	return body
}
