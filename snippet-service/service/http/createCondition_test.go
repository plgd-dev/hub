package http_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
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
)

func makeCreateCondition(id, configurationID, owner string) *snippetPb.Condition {
	return &snippetPb.Condition{
		Id:                 id,
		Name:               snippetTest.ConditionName(0),
		Enabled:            true,
		ConfigurationId:    configurationID,
		Owner:              owner,
		DeviceIdFilter:     snippetTest.ConditionDeviceIdFilter(2, 2),
		ResourceTypeFilter: snippetTest.ConditionResourceTypeFilter(2, 2),
		ResourceHrefFilter: snippetTest.ConditionResourceHrefFilter(2, 2),
		JqExpressionFilter: snippetTest.ConditionJqExpressionFilter(0),
		ApiAccessToken:     snippetTest.ConditionApiAccessToken(0),
	}
}

func TestRequestHandlerCreateCondition(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	shutDown := service.SetUpServices(ctx, t, service.SetUpServicesOAuth|service.SetUpServicesMachine2MachineOAuth)
	defer shutDown()

	snippetCfg := snippetTest.MakeConfig(t)
	_, shutdownHttp := snippetTest.New(t, snippetCfg)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)
	condID1 := uuid.NewString()
	confID1 := uuid.NewString()
	condID2 := uuid.NewString()
	confID2 := uuid.NewString()

	type args struct {
		cond  *snippetPb.Condition
		token string
	}
	tests := []struct {
		name         string
		args         args
		wantHTTPCode int
		want         *snippetPb.Condition
		wantErr      bool
	}{
		{
			name: "create",
			args: args{
				cond:  makeCreateCondition(condID1, confID1, ""),
				token: token,
			},
			wantHTTPCode: http.StatusOK,
			want:         makeCreateCondition(condID1, confID1, oauthService.DeviceUserID),
		},
		{
			name: "create - with owner",
			args: args{
				cond:  makeCreateCondition(condID2, confID2, oauthService.DeviceUserID),
				token: token,
			},
			wantHTTPCode: http.StatusOK,
			want:         makeCreateCondition(condID2, confID2, oauthService.DeviceUserID),
		},
		{
			name: "create - generate ID",
			args: args{
				cond:  makeCreateCondition("", uuid.NewString(), ""),
				token: token,
			},
			wantHTTPCode: http.StatusOK,
			want:         makeCreateCondition("", "", oauthService.DeviceUserID),
		},
		{
			name: "non-matching owner",
			args: args{
				cond:  makeCreateCondition(uuid.NewString(), uuid.NewString(), "non-matching-owner"),
				token: token,
			},
			wantHTTPCode: http.StatusForbidden,
			wantErr:      true,
		},
		{
			name: "duplicit ID",
			args: args{
				cond:  makeCreateCondition(condID1, uuid.NewString(), ""),
				token: token,
			},
			wantHTTPCode: http.StatusInternalServerError,
			wantErr:      true,
		},
		{
			name: "missing configuration id",
			args: args{
				cond:  makeCreateCondition(uuid.NewString(), "", ""),
				token: token,
			},
			wantHTTPCode: http.StatusBadRequest,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := httpTest.GetContentData(&pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data:        test.EncodeToCbor(t, tt.args.cond),
			}, message.AppJSON.String())
			require.NoError(t, err)
			rb := httpTest.NewRequest(http.MethodPost, snippetTest.HTTPURI(snippetHttp.Conditions), bytes.NewReader(data)).AuthToken(token)
			rb = rb.ContentType(message.AppOcfCbor.String())
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
			if tt.want.GetId() == "" {
				tt.want.Id = got.GetId()
			}
			if tt.want.GetConfigurationId() == "" {
				tt.want.ConfigurationId = got.GetConfigurationId()
			}
			snippetTest.CmpCondition(t, tt.want, &got, true)
		})
	}
}
