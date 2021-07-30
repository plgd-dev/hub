package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
)

func TestRequestHandler_CancelDeviceMetadataUpdates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	client, _, devicePendings, shutdown := initPendingEvents(ctx, t)
	defer shutdown()

	require.Equal(t, len(devicePendings), 2)

	type args struct {
		req *pb.CancelDeviceMetadataUpdatesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *pb.CancelResponse
	}{
		{
			name: "cancel one pending",
			args: args{
				req: &pb.CancelDeviceMetadataUpdatesRequest{
					DeviceId:            devicePendings[0].DeviceID,
					CorrelationIdFilter: []string{devicePendings[0].CorrelationID},
				},
			},
			want: &pb.CancelResponse{
				CorrelationIds: []string{devicePendings[0].CorrelationID},
			},
		},
		{
			name: "duplicate cancel event",
			args: args{
				req: &pb.CancelDeviceMetadataUpdatesRequest{
					DeviceId:            devicePendings[0].DeviceID,
					CorrelationIdFilter: []string{devicePendings[0].CorrelationID},
				},
			},
			wantErr: true,
		},
		{
			name: "cancel all events",
			args: args{
				req: &pb.CancelDeviceMetadataUpdatesRequest{
					DeviceId: devicePendings[0].DeviceID,
				},
			},
			want: &pb.CancelResponse{
				CorrelationIds: []string{devicePendings[1].CorrelationID},
			},
		},
	}

	token := oauthTest.GetServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.CancelDeviceMetadataUpdates(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			cmpCancel(t, tt.want, resp)
		})
	}
}
