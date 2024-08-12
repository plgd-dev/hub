package http_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
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

func TestRequestHandlerDeleteAppliedConfigurations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	shutDown := service.SetUpServices(ctx, t, service.SetUpServicesOAuth|service.SetUpServicesMachine2MachineOAuth)
	defer shutDown()

	snippetCfg := test.MakeConfig(t)
	ss, shutdownHttp := test.New(t, snippetCfg)
	defer shutdownHttp()

	_ = test.AddAppliedConfigurationsToStore(ctx, t, ss.SnippetServiceStore())

	conn, err := grpc.NewClient(config.SNIPPET_SERVICE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	ssc := pb.NewSnippetServiceClient(conn)

	getConfigurations := func(ctxWithToken context.Context) map[string]*pb.AppliedConfiguration {
		getClient, errG := ssc.GetAppliedConfigurations(ctxWithToken, &pb.GetAppliedConfigurationsRequest{})
		require.NoError(t, errG)
		defer func() {
			_ = getClient.CloseSend()
		}()
		confs := make(map[string]*pb.AppliedConfiguration)
		for {
			conf, errR := getClient.Recv()
			if errors.Is(errR, io.EOF) {
				break
			}
			require.NoError(t, errR)
			confs[conf.GetId()] = conf
		}
		return confs
	}

	type args struct {
		token    string
		idFilter []string
	}
	tests := []struct {
		name         string
		args         args
		wantHTTPCode int
		wantErr      bool
		want         func()
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
			name: "owner1/all",
			args: args{
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: test.Owner(1),
				}),
			},
			wantHTTPCode: http.StatusOK,
			want: func() {
				token := test.GetTokenWithOwnerClaim(t, test.Owner(1), snippetCfg.APIs.GRPC.Authorization.OwnerClaim)
				ctxWithToken := pkgGrpc.CtxWithToken(ctx, token)
				confs := getConfigurations(ctxWithToken)
				require.Empty(t, confs)
			},
		},
		{
			name: "owner2/id{0,1,2,3,4,5}",
			args: args{
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: test.Owner(2),
				}),
				idFilter: []string{
					test.AppliedConfigurationID(0), test.AppliedConfigurationID(1), test.AppliedConfigurationID(2), test.AppliedConfigurationID(3),
					test.AppliedConfigurationID(4), test.AppliedConfigurationID(5),
				},
			},
			wantHTTPCode: http.StatusOK,
			want: func() {
				token := test.GetTokenWithOwnerClaim(t, test.Owner(2), snippetCfg.APIs.GRPC.Authorization.OwnerClaim)
				ctxWithToken := pkgGrpc.CtxWithToken(ctx, token)
				confs := getConfigurations(ctxWithToken)
				require.NotEmpty(t, confs)
				for _, c := range confs {
					owner := c.GetOwner()
					require.Equal(t, test.Owner(2), owner)
					id := c.GetId()
					if id == test.AppliedConfigurationID(0) || id == test.AppliedConfigurationID(1) || id == test.AppliedConfigurationID(2) ||
						id == test.AppliedConfigurationID(3) || id == test.AppliedConfigurationID(4) || id == test.AppliedConfigurationID(5) {
						require.FailNow(t, "configuration not deleted")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpTest.NewRequest(http.MethodDelete, test.HTTPURI(snippetHttp.AppliedConfigurations), nil).AuthToken(tt.args.token)
			rb = rb.Accept(pkgHttp.ApplicationProtoJsonContentType).ContentType(message.AppCBOR.String()).IDFilter(tt.args.idFilter)
			resp := httpTest.Do(t, rb.Build(ctx, t))
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var deleteResp pb.DeleteAppliedConfigurationsResponse
			err := pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &deleteResp)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want()
		})
	}
}
