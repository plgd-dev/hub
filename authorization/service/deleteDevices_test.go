package service

import (
	"context"
	"testing"

	"github.com/plgd-dev/cloud/authorization/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/stretchr/testify/require"
)

func TestService_DeleteDevice(t *testing.T) {
	const testDevID1 = "testDeviceID1"
	const testDevID2 = "testDeviceID2"
	const testDevID3 = "testDeviceID3"

	type args struct {
		ctx     context.Context
		request *pb.DeleteDevicesRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.DeleteDevicesResponse
		wantErr bool
	}{
		{
			name: "invalid userId",
			args: args{
				request: &pb.DeleteDevicesRequest{},
			},
			wantErr: true,
		},
		{
			name: "invalid deviceId",
			args: args{
				request: &pb.DeleteDevicesRequest{
					UserId: "userId",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid accesstoken",
			args: args{
				request: &pb.DeleteDevicesRequest{
					UserId:    "userId",
					DeviceIds: []string{"deviceId"},
				},
			},
			wantErr: true,
		},
		{
			name: "not belongs to user",
			args: args{
				request: &pb.DeleteDevicesRequest{
					DeviceIds: []string{testDevID1},
					UserId:    "aaa",
				},
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken),
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				request: &pb.DeleteDevicesRequest{
					DeviceIds: []string{testDevID1},
					UserId:    testUserID,
				},
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken),
			},
			want: &pb.DeleteDevicesResponse{DeviceIds: []string{testDevID1}},
		},
		{
			name: "multiple",
			args: args{
				request: &pb.DeleteDevicesRequest{
					DeviceIds: []string{testDevID2, testDevID3},
					UserId:    testUserID,
				},
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken),
			},
			want: &pb.DeleteDevicesResponse{DeviceIds: []string{testDevID2, testDevID3}},
		},
		{
			name: "duplicit",
			args: args{
				request: &pb.DeleteDevicesRequest{
					DeviceIds: []string{testDevID1},
					UserId:    testUserID,
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
	persistDevice(t, s.service.persistence, newTestDeviceWithID(testDevID1))
	persistDevice(t, s.service.persistence, newTestDeviceWithID(testDevID2))
	persistDevice(t, s.service.persistence, newTestDeviceWithID(testDevID3))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.service.DeleteDevices(context.Background(), tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
