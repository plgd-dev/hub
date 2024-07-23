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

func makeUpdateCondition(id, configurationID, owner string, i int) *snippetPb.Condition {
	return &snippetPb.Condition{
		Id:                 id,
		ConfigurationId:    configurationID,
		Owner:              owner,
		Version:            uint64(i),
		Name:               snippetTest.ConditionName(i),
		Enabled:            true,
		DeviceIdFilter:     snippetTest.ConditionDeviceIdFilter(i+1, i+1),
		ResourceTypeFilter: snippetTest.ConditionResourceTypeFilter(i+1, i+1),
		ResourceHrefFilter: snippetTest.ConditionResourceHrefFilter(i+1, i+1),
		JqExpressionFilter: snippetTest.ConditionJqExpressionFilter(i),
		ApiAccessToken:     snippetTest.ConditionApiAccessToken(i),
	}
}

func TestRequestHandlerUpdateCondition(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	shutDown := service.SetUpServices(ctx, t, service.SetUpServicesOAuth|service.SetUpServicesMachine2MachineOAuth)
	defer shutDown()

	_, shutdownHttp := snippetTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)

	conn, err := grpc.NewClient(config.SNIPPET_SERVICE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := snippetPb.NewSnippetServiceClient(conn)

	cond := makeUpdateCondition(uuid.NewString(), uuid.NewString(), oauthService.DeviceUserID, 0)
	_, err = c.CreateCondition(pkgGrpc.CtxWithToken(ctx, token), cond)
	require.NoError(t, err)

	type args struct {
		id    string
		cond  *snippetPb.Condition
		token string
	}
	tests := []struct {
		name         string
		args         args
		want         *snippetPb.Condition
		wantHTTPCode int
		wantErr      bool
	}{
		{
			name: "invalid Id",
			args: args{
				id: "invalid",
				cond: func() *snippetPb.Condition {
					c := makeUpdateCondition(cond.GetId(), cond.GetConfigurationId(), cond.GetOwner(), 1)
					c.Id = "invalid"
					return c
				}(),
				token: token,
			},
			wantHTTPCode: http.StatusBadRequest,
			wantErr:      true,
		},
		{
			name: "invalid configurationID",
			args: args{
				id: cond.GetId(),
				cond: func() *snippetPb.Condition {
					c := makeUpdateCondition(cond.GetId(), cond.GetConfigurationId(), cond.GetOwner(), 1)
					c.ConfigurationId = "invalid"
					return c
				}(),
				token: token,
			},
			wantHTTPCode: http.StatusBadRequest,
			wantErr:      true,
		},
		{
			name: "configurationID mismatch",
			args: args{
				id:    cond.GetId(),
				cond:  makeUpdateCondition(cond.GetId(), uuid.NewString(), oauthService.DeviceUserID, 1),
				token: token,
			},
			wantHTTPCode: http.StatusInternalServerError,
			wantErr:      true,
		},
		{
			name: "non-matching owner",
			args: args{
				id:    cond.GetId(),
				cond:  makeUpdateCondition(cond.GetId(), uuid.NewString(), "non-matching owner", 1),
				token: token,
			},
			wantHTTPCode: http.StatusForbidden,
			wantErr:      true,
		},
		{
			name: "update",
			args: args{
				id:    cond.GetId(),
				cond:  makeUpdateCondition(cond.GetId(), "", "", 1),
				token: token,
			},
			wantHTTPCode: http.StatusOK,
			want:         makeUpdateCondition(cond.GetId(), cond.GetConfigurationId(), oauthService.DeviceUserID, 1),
		},
		{
			name: "update (with owner)",
			args: args{
				id:    cond.GetId(),
				cond:  makeUpdateCondition(cond.GetId(), cond.GetConfigurationId(), oauthService.DeviceUserID, 2),
				token: token,
			},
			wantHTTPCode: http.StatusOK,
			want:         makeUpdateCondition(cond.GetId(), cond.GetConfigurationId(), oauthService.DeviceUserID, 2),
		},
		{
			name: "update (with overwritten ID)",
			args: args{
				id: cond.GetId(),
				// the ID will get overwritten by the ID in the query
				cond:  makeUpdateCondition(uuid.NewString(), "", oauthService.DeviceUserID, 3),
				token: token,
			},
			wantHTTPCode: http.StatusOK,
			want:         makeUpdateCondition(cond.GetId(), cond.GetConfigurationId(), oauthService.DeviceUserID, 3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := httpTest.GetContentData(&pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data:        test.EncodeToCbor(t, tt.args.cond),
			}, message.AppJSON.String())
			require.NoError(t, err)

			rb := httpTest.NewRequest(http.MethodPut, snippetTest.HTTPURI(snippetHttp.AliasConditions), bytes.NewReader(data)).AuthToken(tt.args.token)
			rb.Accept(pkgHttp.ApplicationProtoJsonContentType).ContentType(message.AppJSON.String()).ID(tt.args.id)
			resp := httpTest.Do(t, rb.Build(ctx, t))
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got snippetPb.Condition
			err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			snippetTest.CmpCondition(t, tt.want, &got, true)
		})
	}
}
