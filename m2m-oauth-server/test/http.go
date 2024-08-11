package test

import (
	"context"
	"net/http"
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	"github.com/stretchr/testify/require"
)

func HTTPURI(uri string) string {
	return testHttp.HTTPS_SCHEME + config.M2M_OAUTH_SERVER_HTTP_HOST + uri
}

func DeleteTokens(ctx context.Context, t *testing.T, tokenIDs []string, token string) {
	rb := testHttp.NewRequest(http.MethodDelete, HTTPURI(uri.Tokens), nil).AuthToken(token)
	if len(tokenIDs) > 0 {
		rb.AddQuery(uri.IDFilterQuery, tokenIDs...)
	}
	rb = rb.ContentType(message.AppOcfCbor.String())
	resp := testHttp.Do(t, rb.Build(ctx, t))
	defer func() {
		_ = resp.Body.Close()
	}()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
