package service_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func doPendingCommand(t *testing.T, request *http.Request) (*pb.CancelPendingCommandsResponse, int, error) {
	resp := httpgwTest.HTTPDo(t, request)
	defer func() {
		_ = resp.Body.Close()
	}()
	var v pb.CancelPendingCommandsResponse
	err := pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &v)
	return &v, resp.StatusCode, err
}

func TestRequestHandlerCancelPendingCommands(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	_, resourcePendings, _, shutdown := pbTest.InitPendingEvents(ctx, t)
	defer shutdown()

	require.Len(t, resourcePendings, 4)

	type args struct {
		deviceID            string
		href                string
		correlationIdFilter []string
		accept              string
	}
	tests := []struct {
		name         string
		args         args
		wantErr      bool
		want         *pb.CancelPendingCommandsResponse
		wantHTTPCode int
	}{
		{
			name: "cancel one pending",
			args: args{
				deviceID:            resourcePendings[0].ResourceId.GetDeviceId(),
				href:                resourcePendings[0].ResourceId.GetHref(),
				correlationIdFilter: []string{resourcePendings[0].CorrelationID},
				accept:              pkgHttp.ApplicationProtoJsonContentType,
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{resourcePendings[0].CorrelationID},
			},
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "duplicate cancel event",
			args: args{
				deviceID:            resourcePendings[0].ResourceId.GetDeviceId(),
				href:                resourcePendings[0].ResourceId.GetHref(),
				correlationIdFilter: []string{resourcePendings[0].CorrelationID},
				accept:              pkgHttp.ApplicationProtoJsonContentType,
			},
			wantErr:      true,
			wantHTTPCode: http.StatusNotFound,
		},
		{
			name: "cancel all events",
			args: args{
				deviceID: resourcePendings[0].ResourceId.GetDeviceId(),
				href:     resourcePendings[0].ResourceId.GetHref(),
				accept:   pkgHttp.ApplicationProtoJsonContentType,
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{resourcePendings[1].CorrelationID, resourcePendings[2].CorrelationID, resourcePendings[3].CorrelationID},
			},
			wantHTTPCode: http.StatusOK,
		},
	}

	token := oauthTest.GetDefaultAccessToken(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodDelete, uri.AliasResourcePendingCommands, nil).AuthToken(token).Accept(tt.args.accept)
			rb.DeviceId(tt.args.deviceID).ResourceHref(tt.args.href).AddCorrelationIdFilter(tt.args.correlationIdFilter)
			v, code, err := doPendingCommand(t, rb.Build())
			assert.Equal(t, tt.wantHTTPCode, code)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			pbTest.CmpCancelPendingCmdResponses(t, tt.want, v)
		})
	}
}

func TestRequestHandlerCancelResourceCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	_, resourcePendings, _, shutdown := pbTest.InitPendingEvents(ctx, t)
	defer shutdown()

	require.Len(t, resourcePendings, 4)

	type args struct {
		deviceID      string
		href          string
		correlationID string
		accept        string
	}
	tests := []struct {
		name         string
		args         args
		wantErr      bool
		want         *pb.CancelPendingCommandsResponse
		wantHTTPCode int
	}{
		{
			name: "cancel one pending",
			args: args{
				deviceID:      resourcePendings[0].ResourceId.GetDeviceId(),
				href:          resourcePendings[0].ResourceId.GetHref(),
				correlationID: resourcePendings[0].CorrelationID,
				accept:        pkgHttp.ApplicationProtoJsonContentType,
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{resourcePendings[0].CorrelationID},
			},
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "duplicate cancel event",
			args: args{
				deviceID:      resourcePendings[0].ResourceId.GetDeviceId(),
				href:          resourcePendings[0].ResourceId.GetHref(),
				correlationID: resourcePendings[0].CorrelationID,
				accept:        pkgHttp.ApplicationProtoJsonContentType,
			},
			wantErr:      true,
			wantHTTPCode: http.StatusNotFound,
		},
	}

	token := oauthTest.GetDefaultAccessToken(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodDelete, uri.AliasResourcePendingCommands+"/"+tt.args.correlationID, nil).AuthToken(token).Accept(tt.args.accept)
			rb.DeviceId(tt.args.deviceID).ResourceHref(tt.args.href)
			v, code, err := doPendingCommand(t, rb.Build())
			assert.Equal(t, tt.wantHTTPCode, code)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			pbTest.CmpCancelPendingCmdResponses(t, tt.want, v)
		})
	}
}
