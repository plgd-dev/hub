package service

import (
	"context"
	"testing"

	"github.com/plgd-dev/cloud/authorization/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/stretchr/testify/require"
)

func TestService_DeleteDevice(t *testing.T) {
	type args struct {
		ctx     context.Context
		request *pb.DeleteDeviceRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.DeleteDeviceResponse
		wantErr bool
	}{
		{
			name: "invalid userId",
			args: args{
				request: &pb.DeleteDeviceRequest{},
			},
			wantErr: true,
		},
		{
			name: "invalid deviceId",
			args: args{
				request: &pb.DeleteDeviceRequest{
					UserId: "userId",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid accesstoken",
			args: args{
				request: &pb.DeleteDeviceRequest{
					UserId:   "userId",
					DeviceId: "deviceId",
				},
			},
			wantErr: true,
		},
		{
			name: "not belongs to user",
			args: args{
				request: &pb.DeleteDeviceRequest{
					DeviceId: testDeviceID,
					UserId:   "aaa",
				},
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken),
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				request: &pb.DeleteDeviceRequest{
					DeviceId: testDeviceID,
					UserId:   testUserID,
				},
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken),
			},
			want: &pb.DeleteDeviceResponse{},
		},
		{
			name: "duplicit",
			args: args{
				request: &pb.DeleteDeviceRequest{
					DeviceId: testDeviceID,
					UserId:   testUserID,
				},
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken),
			},
			wantErr: true,
		},
	}
	s, shutdown := newTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	persistDevice(t, s.service.persistence, newTestDevice())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.service.DeleteDevice(context.Background(), tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
