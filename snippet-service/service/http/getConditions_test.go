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
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	snippetHttp "github.com/plgd-dev/hub/v2/snippet-service/service/http"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/snippet-service/uri"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerGetConditions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	shutDown := service.SetUpServices(ctx, t, service.SetUpServicesOAuth|service.SetUpServicesMachine2MachineOAuth)
	defer shutDown()

	snippetCfg := test.MakeConfig(t)
	_, shutdownHttp := test.New(t, snippetCfg)
	defer shutdownHttp()

	conn, err := grpc.NewClient(config.SNIPPET_SERVICE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewSnippetServiceClient(conn)
	n := 30
	conds := test.AddConditions(ctx, t, snippetCfg.APIs.GRPC.Authorization.OwnerClaim, c, n, nil)

	type args struct {
		token        string
		httpIDFilter []string
	}
	tests := []struct {
		name         string
		args         args
		wantHTTPCode int
		wantErr      bool
		want         func(*testing.T, []*pb.Condition)
	}{
		{
			name: "missing owner",
			args: args{
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: nil,
				}),
			},
			wantHTTPCode: http.StatusForbidden,
			wantErr:      true,
		},
		{
			name: "get certain condition",
			args: args{
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: test.Owner(1),
				}),
				httpIDFilter: []string{
					test.ConditionID(1) + "/0",
				},
			},
			wantHTTPCode: http.StatusOK,
			want: func(t *testing.T, values []*pb.Condition) {
				require.Len(t, values, 1)
				require.Equal(t, test.ConditionID(1), values[0].GetId())
				require.Equal(t, uint64(0), values[0].GetVersion())
			},
		},
		{
			name: "owner1/default",
			args: args{
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: test.Owner(1),
				}),
			},
			wantHTTPCode: http.StatusOK,
			want: func(t *testing.T, values []*pb.Condition) {
				require.NotEmpty(t, values)
				require.InDelta(t, test.RuntimeConfig.NumConditions/test.RuntimeConfig.NumOwners, len(values), 1)
				for _, v := range values {
					cond, ok := conds[v.GetId()]
					require.True(t, ok)
					test.ConditionContains(t, cond, v)
				}
			},
		},
		{
			name: "owner1/all",
			args: args{
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: test.Owner(1),
				}),
				httpIDFilter: []string{
					"//all",
				},
			},
			wantHTTPCode: http.StatusOK,
			want: func(t *testing.T, values []*pb.Condition) {
				require.NotEmpty(t, values)
				require.InDelta(t, n/test.RuntimeConfig.NumOwners, len(values), 1)
				for _, v := range values {
					cond, ok := conds[v.GetId()]
					require.True(t, ok)
					test.ConditionContains(t, cond, v)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpTest.NewRequest(http.MethodGet, test.HTTPURI(snippetHttp.Conditions), nil).AuthToken(tt.args.token)
			rb = rb.Accept(pkgHttp.ApplicationProtoJsonContentType).ContentType(message.AppCBOR.String())
			if len(tt.args.httpIDFilter) > 0 {
				rb = rb.AddQuery(uri.HTTPIDFilterQueryKey, tt.args.httpIDFilter...)
			}
			resp := httpTest.Do(t, rb.Build(ctx, t))
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			values := make([]*pb.Condition, 0, 1)
			for {
				var value pb.Condition
				err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &value)
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
