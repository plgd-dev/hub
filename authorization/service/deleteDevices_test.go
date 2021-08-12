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
	const testDevID4 = "testDeviceID4"
	const testUser2 = "testUser2"
	const testUser2DevID1 = "test2DeviceID1"
	const testUser2DevID2 = "test2DeviceID2"
	const testUser2DevID3 = "test2DeviceID3"

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
			want: &pb.DeleteDevicesResponse{},
		},
		{
			name: "invalid accesstoken",
			args: args{
				request: &pb.DeleteDevicesRequest{
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
			want: &pb.DeleteDevicesResponse{},
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
			want: &pb.DeleteDevicesResponse{},
		},
		{
			name: "owned and not owned",
			args: args{
				request: &pb.DeleteDevicesRequest{
					DeviceIds: []string{testDevID4, testUser2DevID1},
					UserId:    testUserID,
				},
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken),
			},
			want: &pb.DeleteDevicesResponse{DeviceIds: []string{testDevID4}},
		},
		{
			name: "all owned by testUser2",
			args: args{
				request: &pb.DeleteDevicesRequest{
					DeviceIds: nil,
					UserId:    testUser2,
				},
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), testAccessToken),
			},
			want: &pb.DeleteDevicesResponse{DeviceIds: []string{testUser2DevID1, testUser2DevID2, testUser2DevID3}},
		},
	}
	s, shutdown := newTestService(t)
	defer shutdown()
	defer func() {
		err := s.cleanUp()
		require.NoError(t, err)
	}()
	persistDevice(t, s.service.persistence, newTestDeviceWithIDAndOwner(testDevID1, testUserID))
	persistDevice(t, s.service.persistence, newTestDeviceWithIDAndOwner(testDevID2, testUserID))
	persistDevice(t, s.service.persistence, newTestDeviceWithIDAndOwner(testDevID3, testUserID))
	persistDevice(t, s.service.persistence, newTestDeviceWithIDAndOwner(testDevID4, testUserID))
	persistDevice(t, s.service.persistence, newTestDeviceWithIDAndOwner(testUser2DevID1, testUser2))
	persistDevice(t, s.service.persistence, newTestDeviceWithIDAndOwner(testUser2DevID2, testUser2))
	persistDevice(t, s.service.persistence, newTestDeviceWithIDAndOwner(testUser2DevID3, testUser2))

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
