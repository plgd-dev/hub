package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/stretchr/testify/require"
)

func TestRequestHandlerCancelPendingCommands(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	client, resourcePendings, _, shutdown := pbTest.InitPendingEvents(ctx, t)
	defer shutdown()

	require.Len(t, resourcePendings, 4)

	type args struct {
		req *pb.CancelPendingCommandsRequest
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
				req: &pb.CancelPendingCommandsRequest{
					ResourceId:          resourcePendings[0].ResourceId,
					CorrelationIdFilter: []string{resourcePendings[0].CorrelationID},
				},
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{resourcePendings[0].CorrelationID},
			},
		},
		{
			name: "duplicate cancel event",
			args: args{
				req: &pb.CancelPendingCommandsRequest{
					ResourceId:          resourcePendings[0].ResourceId,
					CorrelationIdFilter: []string{resourcePendings[0].CorrelationID},
				},
			},
			wantErr: true,
		},
		{
			name: "cancel all events",
			args: args{
				req: &pb.CancelPendingCommandsRequest{
					ResourceId: resourcePendings[0].ResourceId,
				},
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{resourcePendings[1].CorrelationID, resourcePendings[2].CorrelationID, resourcePendings[3].CorrelationID},
			},
		},
	}

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.CancelPendingCommands(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			pbTest.CmpCancelPendingCmdResponses(t, tt.want, resp)
		})
	}
}
