package test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	grpcPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	"github.com/stretchr/testify/require"
)

func HTTPURI(uri string) string {
	return testHttp.HTTPS_SCHEME + config.M2M_OAUTH_SERVER_HTTP_HOST + uri
}

func BlacklistTokens(ctx context.Context, t *testing.T, tokenIDs []string, token string) {
	data, err := testHttp.GetContentData(&grpcPb.Content{
		ContentType: message.AppOcfCbor.String(),
		Data: test.EncodeToCbor(t, &pb.BlacklistTokensRequest{
			IdFilter: tokenIDs,
		}),
	}, message.AppJSON.String())
	require.NoError(t, err)
	rb := testHttp.NewRequest(http.MethodPost, HTTPURI(uri.BlacklistTokens), bytes.NewReader(data)).AuthToken(token)
	rb = rb.ContentType(message.AppOcfCbor.String())
	resp := testHttp.Do(t, rb.Build(ctx, t))
	defer func() {
		_ = resp.Body.Close()
	}()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
