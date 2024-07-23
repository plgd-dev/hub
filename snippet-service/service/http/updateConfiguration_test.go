package http_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerUpdateConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	shutDown := service.SetUpServices(ctx, t, service.SetUpServicesOAuth|service.SetUpServicesMachine2MachineOAuth)
	defer shutDown()

	_, shutdownHttp := snippetTest.SetUp(t)
	defer shutdownHttp()

	conn, err := grpc.NewClient(config.SNIPPET_SERVICE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := snippetPb.NewSnippetServiceClient(conn)
	conf := &snippetPb.Configuration{
		Id:      uuid.NewString(),
		Version: 0,
		Name:    "configurationToUpdate",
		Resources: []*snippetPb.Configuration_Resource{
			makeTestResource(t, "/test/1", 1),
		},
	}
	token := oauthTest.GetDefaultAccessToken(t)
	_, err = c.CreateConfiguration(pkgGrpc.CtxWithToken(ctx, token), conf)
	require.NoError(t, err)

	type args struct {
		id    string
		conf  *snippetPb.Configuration
		token string
	}
	tests := []struct {
		name         string
		args         args
		wantHTTPCode int
		wantErr      bool
	}{
		{
			name: "invalid ID",
			args: args{
				id: "invalid",
				conf: &snippetPb.Configuration{
					Version: 42,
					Resources: []*snippetPb.Configuration_Resource{
						makeTestResource(t, "/test/1", 42),
					},
				},
				token: token,
			},
			wantHTTPCode: http.StatusBadRequest,
			wantErr:      true,
		},
		{
			name: "missing resources",
			args: args{
				id: conf.GetId(),
				conf: &snippetPb.Configuration{
					Version: 42,
				},
				token: token,
			},
			wantHTTPCode: http.StatusBadRequest,
			wantErr:      true,
		},
		{
			name: "non-matching owner",
			args: args{
				id: conf.GetId(),
				conf: &snippetPb.Configuration{
					Version: 42,
					Owner:   "non-matching owner",
					Resources: []*snippetPb.Configuration_Resource{
						makeTestResource(t, "/test/2", 52),
					},
				},
				token: token,
			},
			wantHTTPCode: http.StatusForbidden,
			wantErr:      true,
		},
		{
			name: "update",
			args: args{
				id: conf.GetId(),
				conf: &snippetPb.Configuration{
					Version: 1,
					Name:    "updated1",
					Resources: []*snippetPb.Configuration_Resource{
						makeTestResource(t, "/test/3", 62),
						makeTestResource(t, "/test/4", 72),
					},
				},
				token: token,
			},
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "update (with owner)",
			args: args{
				id: conf.GetId(),
				conf: &snippetPb.Configuration{
					Version: 2,
					Owner:   oauthService.DeviceUserID,
					Name:    "updated2",
					Resources: []*snippetPb.Configuration_Resource{
						makeTestResource(t, "/test/5", 82),
					},
				},
				token: token,
			},
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "update (with overwritten ID)",
			args: args{
				id: conf.GetId(),
				conf: &snippetPb.Configuration{
					Id:      uuid.NewString(), // this ID will get overwritten by the ID in the query
					Version: 3,
					Name:    "updated3",
					Resources: []*snippetPb.Configuration_Resource{
						makeTestResource(t, "/test/6", 92),
						makeTestResource(t, "/test/7", 102),
					},
				},
				token: token,
			},
			wantHTTPCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := httpTest.GetContentData(&pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data:        test.EncodeToCbor(t, tt.args.conf),
			}, message.AppJSON.String())
			require.NoError(t, err)

			rb := httpTest.NewRequest(http.MethodPut, snippetTest.HTTPURI(snippetHttp.AliasConfigurations), bytes.NewReader(data)).AuthToken(tt.args.token)
			rb.Accept(pkgHttp.ApplicationProtoJsonContentType).ContentType(message.AppJSON.String()).ID(tt.args.id)
			resp := httpTest.Do(t, rb.Build(ctx, t))
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got snippetPb.Configuration
			err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			want := tt.args.conf
			want.Id = tt.args.id
			want.Owner = oauthService.DeviceUserID
			snippetTest.CmpConfiguration(t, want, &got, true)
		})
	}
}
