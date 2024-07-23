package http_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	grpcPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	oauthsigner "github.com/plgd-dev/hub/v2/m2m-oauth-server/oauthSigner"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	m2mOauthServerTest "github.com/plgd-dev/hub/v2/m2m-oauth-server/test"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestCreateToken(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	tearDown := testService.SetUp(ctx, t)
	defer tearDown()

	type args struct {
		req *pb.CreateTokenRequest
	}
	tests := []struct {
		name         string
		args         args
		wantHTTPCode int
		want         *pb.CreateTokenResponse
		wantErr      bool
	}{
		{
			name: "create token",
			args: args{
				req: &pb.CreateTokenRequest{
					ClientId:     m2mOauthServerTest.ServiceOAuthClient.ID,
					ClientSecret: m2mOauthServerTest.GetSecret(t, m2mOauthServerTest.ServiceOAuthClient.ID),
					GrantType:    string(oauthsigner.GrantTypeClientCredentials),
					TokenName:    "service token",
				},
			},
			wantHTTPCode: http.StatusOK,
			want: &pb.CreateTokenResponse{
				TokenType: "Bearer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := testHttp.GetContentData(&grpcPb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data:        test.EncodeToCbor(t, tt.args.req),
			}, message.AppJSON.String())
			require.NoError(t, err)
			rb := testHttp.NewRequest(http.MethodPost, m2mOauthServerTest.HTTPURI(uri.Tokens), bytes.NewReader(data))
			rb = rb.ContentType(message.AppOcfCbor.String())
			resp := testHttp.Do(t, rb.Build(ctx, t))
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got pb.CreateTokenResponse
			err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got.GetAccessToken())
			require.Equal(t, tt.want.GetTokenType(), got.GetTokenType())
		})
	}
}
