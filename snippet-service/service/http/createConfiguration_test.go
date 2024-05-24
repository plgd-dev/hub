package http_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	snippetPb "github.com/plgd-dev/hub/v2/snippet-service/pb"
	snippetHttp "github.com/plgd-dev/hub/v2/snippet-service/service/http"
	snippetTest "github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func HTTPURI(uri string) string {
	return httpTest.HTTPS_SCHEME + config.SNIPPET_SERVICE_HTTP_HOST + uri
}

func TestRequestHandlerCreateConfiguration(t *testing.T) {
	type args struct {
		accept string
		conf   *snippetPb.Configuration
	}
	tests := []struct {
		name         string
		args         args
		wantData     map[string]interface{}
		wantHTTPCode int
	}{
		{
			name: "create",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				conf: &snippetPb.Configuration{
					Id:    uuid.New().String(),
					Owner: "owner",
					Name:  "first",
					Resources: []*snippetPb.Configuration_Resource{
						{
							Href: "/test/1",
							Content: &commands.Content{
								Data: test.EncodeToCbor(t, map[string]interface{}{
									"power": 42,
								}),
								ContentType:       message.AppOcfCbor.String(),
								CoapContentFormat: int32(message.AppOcfCbor),
							},
							TimeToLive: 60,
						},
					},
				},
			},
			wantHTTPCode: http.StatusOK,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	shutDown := service.SetUpServices(context.Background(), t, service.SetUpServicesOAuth)
	defer shutDown()

	shutdownHttp := snippetTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := httpTest.GetContentData(&pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data:        test.EncodeToCbor(t, tt.args.conf),
			}, message.AppJSON.String())
			require.NoError(t, err)

			rb := httpTest.NewRequest(http.MethodPost, HTTPURI(snippetHttp.Configurations), bytes.NewReader(data)).AuthToken(token)
			rb.Accept(tt.args.accept).ContentType(message.AppJSON.String())
			resp := httpTest.Do(t, rb.Build(ctx, t))
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got snippetPb.Configuration
			err = httpTest.Unmarshal(resp.StatusCode, resp.Body, &got)
			require.NoError(t, err)

			snippetTest.CmpConfiguration(t, tt.args.conf, &got)
		})
	}
}
