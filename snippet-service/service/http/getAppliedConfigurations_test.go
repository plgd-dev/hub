package http_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	snippetHttp "github.com/plgd-dev/hub/v2/snippet-service/service/http"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/snippet-service/uri"
	"github.com/plgd-dev/hub/v2/test/config"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestRequestHandlerGetAppliedConfigurations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	shutDown := service.SetUpServices(context.Background(), t, service.SetUpServicesOAuth)
	defer shutDown()

	snippetCfg := test.MakeConfig(t)
	logCfg := log.MakeDefaultConfig()
	logCfg.Level = log.DebugLevel
	log.Setup(logCfg)
	snippetCfg.Log = logCfg
	ss, shutdownHttp := test.New(t, snippetCfg)
	defer shutdownHttp()

	appliedConfs := test.AddAppliedConfigurations(ctx, t, snippetCfg.APIs.GRPC.Authorization.OwnerClaim, ss)

	type args struct {
		token                     string
		idFilter                  []string
		httpConfigurationIdFilter []string
		httpConditionIdFilter     []string
	}
	tests := []struct {
		name         string
		args         args
		wantHTTPCode int
		wantErr      bool
		want         func([]*pb.AppliedDeviceConfiguration)
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
			name: "owner1",
			args: args{
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: test.Owner(1),
				}),
			},
			wantHTTPCode: http.StatusOK,
			want: func(values []*pb.AppliedDeviceConfiguration) {
				require.NotEmpty(t, values)
				for _, v := range values {
					conf, ok := appliedConfs[v.GetId()]
					require.True(t, ok)
					test.CmpAppliedDeviceConfiguration(t, conf, v, false)
				}
			},
		},
		/* TODO need to be implemented in mongodb
		{
			name: "get certain applied configuration",
			args: args{
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: test.Owner(0),
				}),
				httpConfigurationIdFilter: []string{
					test.ConfigurationID(0) + "/0",
				},
			},
			wantHTTPCode: http.StatusOK,
			want: func(values []*pb.AppliedDeviceConfiguration) {
				require.Len(t, values, 1)
				require.Equal(t, test.ConfigurationID(0), values[0].GetConfigurationId().GetId())
				require.Equal(t, uint64(0), values[0].GetConfigurationId().GetVersion())
			},
		},
		{
			name: "get certain applied condition",
			args: args{
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: test.Owner(0),
				}),
				httpConditionIdFilter: []string{
					test.ConditionID(0) + "/0",
				},
			},
			wantHTTPCode: http.StatusOK,
			want: func(values []*pb.AppliedDeviceConfiguration) {
				require.Len(t, values, 1)
				require.Equal(t, test.ConditionID(0), values[0].GetConditionId().GetId())
				require.Equal(t, uint64(0), values[0].GetConditionId().GetVersion())
			},
		},
		*/
		{
			name: "owner0/id{0,1,2,3,4,5}",
			args: args{
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: test.Owner(0),
				}),
				idFilter: []string{
					test.AppliedConfigurationID(0), test.AppliedConfigurationID(1), test.AppliedConfigurationID(2), test.AppliedConfigurationID(3),
					test.AppliedConfigurationID(4), test.AppliedConfigurationID(5),
				},
			},
			wantHTTPCode: http.StatusOK,
			want: func(values []*pb.AppliedDeviceConfiguration) {
				require.NotEmpty(t, values)
				for _, v := range values {
					require.Equal(t, test.Owner(0), v.GetOwner())
					conf, ok := appliedConfs[v.GetId()]
					require.True(t, ok)
					test.CmpAppliedDeviceConfiguration(t, conf, v, false)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpTest.NewRequest(http.MethodGet, test.HTTPURI(snippetHttp.AppliedConfigurations), nil).AuthToken(tt.args.token)
			rb = rb.Accept(pkgHttp.ApplicationProtoJsonContentType).ContentType(message.AppCBOR.String()).IDFilter(tt.args.idFilter)
			if len(tt.args.httpConfigurationIdFilter) > 0 {
				rb = rb.AddQuery(uri.HTTPConfigurationIDFilter, tt.args.httpConfigurationIdFilter...)
			}
			if len(tt.args.httpConditionIdFilter) > 0 {
				rb = rb.AddQuery(uri.HTTPConditionIDFilter, tt.args.httpConditionIdFilter...)
			}
			resp := httpTest.Do(t, rb.Build(ctx, t))
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			values := make([]*pb.AppliedDeviceConfiguration, 0, 1)
			for {
				var value pb.AppliedDeviceConfiguration
				err := httpTest.Unmarshal(resp.StatusCode, resp.Body, &value)
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
			tt.want(values)
		})
	}
}
