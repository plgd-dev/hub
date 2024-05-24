package http_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	snippetPb "github.com/plgd-dev/hub/v2/snippet-service/pb"
	snippetHttp "github.com/plgd-dev/hub/v2/snippet-service/service/http"
	snippetTest "github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func makeTestResource(t *testing.T, href string, power int) *snippetPb.Configuration_Resource {
	return &snippetPb.Configuration_Resource{
		Href: href,
		Content: &commands.Content{
			Data:              test.EncodeToCbor(t, map[string]interface{}{"power": power}),
			ContentType:       message.AppOcfCbor.String(),
			CoapContentFormat: int32(message.AppOcfCbor),
		},
		TimeToLive: 60,
	}
}

func TestRequestHandlerCreateConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	shutDown := service.SetUpServices(ctx, t, service.SetUpServicesOAuth)
	defer shutDown()

	snippetCfg := snippetTest.MakeConfig(t)
	shutdownHttp := snippetTest.New(t, snippetCfg)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)
	confID1 := uuid.NewString()

	type args struct {
		accept string
		conf   *snippetPb.Configuration
		token  string
	}
	tests := []struct {
		name         string
		args         args
		wantHTTPCode int
		wantErr      bool
	}{
		{
			name: "create",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				conf: &snippetPb.Configuration{
					Id:   confID1,
					Name: "first",
					Resources: []*snippetPb.Configuration_Resource{
						makeTestResource(t, "/test/1", 41),
					},
				},
				token: token,
			},
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "create (with owner)",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				conf: &snippetPb.Configuration{
					Id:    uuid.New().String(),
					Owner: oauthService.DeviceUserID,
					Name:  "second",
					Resources: []*snippetPb.Configuration_Resource{
						makeTestResource(t, "/test/2", 42),
					},
				},
				token: token,
			},
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "non-matching owner",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				conf: &snippetPb.Configuration{
					Id:    uuid.New().String(),
					Owner: "non-matching-owner",
					Name:  "third",
					Resources: []*snippetPb.Configuration_Resource{
						makeTestResource(t, "/test/3", 43),
					},
				},
				token: token,
			},
			wantHTTPCode: http.StatusForbidden,
			wantErr:      true,
		},
		{
			name: "missing id",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				conf: &snippetPb.Configuration{
					Name: "fourth",
					Resources: []*snippetPb.Configuration_Resource{
						makeTestResource(t, "/test/4", 44),
					},
				},
				token: token,
			},
			wantHTTPCode: http.StatusInternalServerError,
			wantErr:      true,
		},
		{
			name: "duplicit ID",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				conf: &snippetPb.Configuration{
					Id:   confID1,
					Name: "fifth",
					Resources: []*snippetPb.Configuration_Resource{
						makeTestResource(t, "/test/5", 45),
					},
				},
				token: token,
			},
			wantHTTPCode: http.StatusInternalServerError,
			wantErr:      true,
		},
		{
			name: "missing resources",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				conf: &snippetPb.Configuration{
					Id:   uuid.New().String(),
					Name: "fifth",
				},
				token: token,
			},
			wantHTTPCode: http.StatusInternalServerError,
			wantErr:      true,
		},
		{
			name: "missing owner in token",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				conf: &snippetPb.Configuration{
					Id:   uuid.New().String(),
					Name: "sixth",
					Resources: []*snippetPb.Configuration_Resource{
						makeTestResource(t, "/test/6", 46),
					},
				},
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: nil,
				}),
			},
			wantHTTPCode: http.StatusForbidden,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := httpTest.GetContentData(&pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data:        test.EncodeToCbor(t, tt.args.conf),
			}, message.AppJSON.String())
			require.NoError(t, err)

			rb := httpTest.NewRequest(http.MethodPost, snippetTest.HTTPURI(snippetHttp.Configurations), bytes.NewReader(data)).AuthToken(tt.args.token)
			rb.Accept(tt.args.accept).ContentType(message.AppJSON.String())
			resp := httpTest.Do(t, rb.Build(ctx, t))
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got snippetPb.Configuration
			err = httpTest.Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			want := tt.args.conf
			want.Owner = oauthService.DeviceUserID
			snippetTest.CmpConfiguration(t, want, &got)
		})
	}
}
