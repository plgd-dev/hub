package service

import (
	"context"
	"testing"

	"github.com/plgd-dev/cloud/authorization/pb"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/stretchr/testify/require"
)

func TestService_RemoveDevice(t *testing.T) {
	type args struct {
		ctx     context.Context
		request *pb.RemoveDeviceRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.RemoveDeviceResponse
		wantErr bool
	}{
		{
			name: "invalid userId",
			args: args{
				request: &pb.RemoveDeviceRequest{},
			},
			wantErr: true,
		},
		{
			name: "invalid deviceId",
			args: args{
				request: &pb.RemoveDeviceRequest{
					UserId: "userId",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid accesstoken",
			args: args{
				request: &pb.RemoveDeviceRequest{
					UserId:   "userId",
					DeviceId: "deviceId",
				},
			},
			wantErr: true,
		},
		{
			name: "not belongs to user",
			args: args{
				request: &pb.RemoveDeviceRequest{
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
				request: &pb.RemoveDeviceRequest{
					DeviceId: testDeviceID,
					UserId:   testUserID,
				},
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken),
			},
			want: &pb.RemoveDeviceResponse{},
		},
		{
			name: "duplicit",
			args: args{
				request: &pb.RemoveDeviceRequest{
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
	defer s.cleanUp()
	persistDevice(t, s.service.persistence, newTestDevice())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.service.RemoveDevice(context.Background(), tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
