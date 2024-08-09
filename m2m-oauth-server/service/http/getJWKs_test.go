package http_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

func TestRequestHandlerGetJWKs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	webTearDown := testService.SetUp(ctx, t)
	defer webTearDown()

	getJWKs(ctx, t)
}

func getJWKs(ctx context.Context, t *testing.T) map[string]interface{} {
	getReq := testHttp.NewRequest(http.MethodGet, testHttp.HTTPS_SCHEME+config.M2M_OAUTH_SERVER_HTTP_HOST+uri.JWKs, nil).Build(ctx, t)
	res := testHttp.Do(t, getReq)
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
