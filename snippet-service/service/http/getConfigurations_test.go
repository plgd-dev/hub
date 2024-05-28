package http_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	snippetHttp "github.com/plgd-dev/hub/v2/snippet-service/service/http"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerGetConfigurations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	shutDown := service.SetUpServices(context.Background(), t, service.SetUpServicesOAuth)
	defer shutDown()

	snippetCfg := test.MakeConfig(t)
	shutdownHttp := test.New(t, snippetCfg)
	defer shutdownHttp()

	conn, err := grpc.NewClient(config.SNIPPET_SERVICE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewSnippetServiceClient(conn)
	confs := test.AddConfigurations(ctx, t, snippetCfg.APIs.GRPC.Authorization.OwnerClaim, c, 30, nil)

	type args struct {
		accept string
		token  string
	}
	tests := []struct {
		name         string
		args         args
		wantHTTPCode int
		wantErr      bool
		want         func(*testing.T, []*pb.Configuration)
	}{
		{
			name: "owner1/all",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: test.ConfigurationOwner(1),
				}),
			},
			wantHTTPCode: http.StatusOK,
			want: func(t *testing.T, values []*pb.Configuration) {
				require.NotEmpty(t, values)
				for _, v := range values {
					conf, ok := confs[v.GetId()]
					require.True(t, ok)
					test.ConfigurationContains(t, conf, v)
				}
			},
		},
		{
			name: "missing owner",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
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
			rb := httpTest.NewRequest(http.MethodGet, test.HTTPURI(snippetHttp.Configurations), nil).AuthToken(tt.args.token)
			rb = rb.Accept(tt.args.accept).ContentType(message.AppCBOR.String())
			resp := httpTest.Do(t, rb.Build(ctx, t))
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			if tt.wantErr {
				return
			}

			values := make([]*pb.Configuration, 0, 1)
			for {
				var value pb.Configuration
				err = httpTest.Unmarshal(resp.StatusCode, resp.Body, &value)
				if errors.Is(err, io.EOF) {
					break
				}
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				values = append(values, &value)
			}
			tt.want(t, values)
		})
	}
}
