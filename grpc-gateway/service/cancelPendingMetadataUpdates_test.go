package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/test/pb"
	"github.com/stretchr/testify/require"
)

func TestRequestHandlerCancelPendingMetadataUpdates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	client, _, devicePendings, shutdown := pbTest.InitPendingEvents(ctx, t)
	defer shutdown()

	require.Equal(t, len(devicePendings), 2)

	type args struct {
		req *pb.CancelPendingMetadataUpdatesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *pb.CancelPendingCommandsResponse
	}{
		{
			name: "cancel one pending",
			args: args{
				req: &pb.CancelPendingMetadataUpdatesRequest{
					DeviceId:            devicePendings[0].DeviceID,
					CorrelationIdFilter: []string{devicePendings[0].CorrelationID},
				},
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{devicePendings[0].CorrelationID},
			},
		},
		{
			name: "duplicate cancel event",
			args: args{
				req: &pb.CancelPendingMetadataUpdatesRequest{
					DeviceId:            devicePendings[0].DeviceID,
					CorrelationIdFilter: []string{devicePendings[0].CorrelationID},
				},
			},
			wantErr: true,
		},
		{
			name: "cancel all events",
			args: args{
				req: &pb.CancelPendingMetadataUpdatesRequest{
					DeviceId: devicePendings[0].DeviceID,
				},
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{devicePendings[1].CorrelationID},
			},
		},
	}

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.CancelPendingMetadataUpdates(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			pbTest.CmpCancelPendingCmdResponses(t, tt.want, resp)
		})
	}
}
