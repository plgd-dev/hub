package http_test

import (
	"context"
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
	"github.com/plgd-dev/hub/v2/test/config"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestRequestHandlerGetAppliedConfigurations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	shutDown := service.SetUpServices(ctx, t, service.SetUpServicesOAuth|service.SetUpServicesMachine2MachineOAuth)
	defer shutDown()

	snippetCfg := test.MakeConfig(t)
	ss, shutdownHttp := test.New(t, snippetCfg)
	defer shutdownHttp()

	appliedConfs := test.AddAppliedConfigurationsToStore(ctx, t, ss.SnippetServiceStore())

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
		want         func(*pb.AppliedConfiguration) bool
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
			want: func(ac *pb.AppliedConfiguration) bool {
				return ac.GetOwner() == test.Owner(1)
			},
		},
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
			want: func(ac *pb.AppliedConfiguration) bool {
				acID := ac.GetId()
				return ac.GetOwner() == test.Owner(0) &&
					(acID == test.AppliedConfigurationID(0) || acID == test.AppliedConfigurationID(1) ||
						acID == test.AppliedConfigurationID(2) || acID == test.AppliedConfigurationID(3) ||
						acID == test.AppliedConfigurationID(4) || acID == test.AppliedConfigurationID(5))
			},
		},
		{
			name: "owner2/configurationId[{id:2,version:2},id:5]",
			args: args{
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: test.Owner(2),
				}),
				httpConfigurationIdFilter: []string{
					test.ConfigurationID(2) + "/2",
					test.ConfigurationID(5) + "/all",
				},
			},
			wantHTTPCode: http.StatusOK,
			want: func(ac *pb.AppliedConfiguration) bool {
				acConfID := ac.GetConfigurationId().GetId()
				acConfVersion := ac.GetConfigurationId().GetVersion()
				return ac.GetOwner() == test.Owner(2) &&
					((acConfID == test.ConfigurationID(2) && acConfVersion == 2) ||
						(acConfID == test.ConfigurationID(5)))
			},
		},
		{
			name: "owner0/conditionId[{id:3,version:3},{id:6},{version:13}]",
			args: args{
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTest, map[string]interface{}{
					snippetCfg.APIs.GRPC.Authorization.OwnerClaim: test.Owner(0),
				}),
				httpConditionIdFilter: []string{
					test.ConditionID(3) + "/3",
					test.ConditionID(6) + "/all",
					"13",
				},
			},
			wantHTTPCode: http.StatusOK,
			want: func(ac *pb.AppliedConfiguration) bool {
				acCondID := ac.GetConditionId().GetId()
				acCondVersion := ac.GetConditionId().GetVersion()
				return ac.GetOwner() == test.Owner(0) &&
					((acCondID == test.ConditionID(3) && acCondVersion == 3) ||
						(acCondID == test.ConditionID(6)) ||
						(acCondVersion == 13))
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

			receivedConfs := make(map[string]*pb.AppliedConfiguration)
			for {
				var value pb.AppliedConfiguration
				err := pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &value)
				if errors.Is(err, io.EOF) {
					break
				}
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				receivedConfs[value.GetId()] = &value
			}

			stored := make(map[string]*pb.AppliedConfiguration)
			for _, ac := range appliedConfs {
				if tt.want(ac.GetAppliedConfiguration()) {
					stored[ac.GetId()] = ac.GetAppliedConfiguration().Clone()
				}
			}
			require.Len(t, receivedConfs, len(stored))
			for _, c := range receivedConfs {
				ac, ok := stored[c.GetId()]
				require.True(t, ok)
				test.CmpAppliedDeviceConfiguration(t, ac, c, false)
			}
		})
	}
}
