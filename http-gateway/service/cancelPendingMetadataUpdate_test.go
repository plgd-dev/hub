package service_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestHandlerCancelDeviceMetadataUpdate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	_, _, devicePendings, shutdown := pbTest.InitPendingEvents(ctx, t)
	defer shutdown()

	require.Equal(t, len(devicePendings), 2)

	type args struct {
		deviceID      string
		correlationId string
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
				deviceID:      devicePendings[0].DeviceID,
				correlationId: devicePendings[0].CorrelationID,
				accept:        uri.ApplicationProtoJsonContentType,
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{devicePendings[0].CorrelationID},
			},
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "duplicate cancel event",
			args: args{
				deviceID:      devicePendings[0].DeviceID,
				correlationId: devicePendings[0].CorrelationID,
				accept:        uri.ApplicationProtoJsonContentType,
			},
			wantErr:      true,
			wantHTTPCode: http.StatusNotFound,
		},
	}

	token := oauthTest.GetDefaultAccessToken(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodDelete, uri.AliasDevicePendingMetadataUpdates+"/"+tt.args.correlationId, nil).AuthToken(token).Accept(tt.args.accept)
			rb.DeviceId(tt.args.deviceID)
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
