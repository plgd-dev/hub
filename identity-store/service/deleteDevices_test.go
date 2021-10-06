package service

import (
	"context"
	"sort"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/plgd-dev/cloud/v2/identity-store/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"github.com/plgd-dev/cloud/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestServiceDeleteDevices(t *testing.T) {
	const testDevID1 = "testDeviceID1"
	const testDevID2 = "testDeviceID2"
	const testDevID3 = "testDeviceID3"
	const testDevID4 = "testDeviceID4"
	jwtWithSubUserId := config.CreateJwtToken(t, jwt.MapClaims{
		"sub": "userId",
	})
	jwtWithSubTestUserID := config.CreateJwtToken(t, jwt.MapClaims{
		"sub": testUserID,
	})
	jwtWithSubAaa := config.CreateJwtToken(t, jwt.MapClaims{
		"sub": "aaa",
	})
	var jwtWithSubTestUser2 = config.CreateJwtToken(t, jwt.MapClaims{
		"sub": testUser2,
	})
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
			name: "invalid deviceId",
			args: args{
				ctx:     kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserId),
				request: &pb.DeleteDevicesRequest{},
			},
			want: &pb.DeleteDevicesResponse{},
		},
		{
			name: "invalid accesstoken",
			args: args{
				ctx: context.Background(),
				request: &pb.DeleteDevicesRequest{
					DeviceIds: []string{"deviceId"},
				},
			},
			wantErr: true,
		},
		{
			name: "not belongs to user",
			args: args{
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubAaa),
				request: &pb.DeleteDevicesRequest{
					DeviceIds: []string{testDevID1},
				},
			},
			want: &pb.DeleteDevicesResponse{},
		},
		{
			name: "valid",
			args: args{
				request: &pb.DeleteDevicesRequest{
					DeviceIds: []string{testDevID1},
				},
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubTestUserID),
			},
			want: &pb.DeleteDevicesResponse{DeviceIds: []string{testDevID1}},
		},
		{
			name: "multiple",
			args: args{
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubTestUserID),
				request: &pb.DeleteDevicesRequest{
					DeviceIds: []string{testDevID2, testDevID3},
				},
			},
			want: &pb.DeleteDevicesResponse{DeviceIds: []string{testDevID2, testDevID3}},
		},
		{
			name: "duplicit",
			args: args{
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubTestUserID),
				request: &pb.DeleteDevicesRequest{
					DeviceIds: []string{testDevID1},
				},
			},
			want: &pb.DeleteDevicesResponse{},
		},
		{
			name: "owned and not owned",
			args: args{
				request: &pb.DeleteDevicesRequest{
					DeviceIds: []string{testDevID4, testUser2DevID1},
				},
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubTestUserID),
			},
			want: &pb.DeleteDevicesResponse{DeviceIds: []string{testDevID4}},
		},
		{
			name: "all owned by testUser2",
			args: args{
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubTestUser2),
				request: &pb.DeleteDevicesRequest{
					DeviceIds: nil,
				},
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
			got, err := s.service.DeleteDevices(tt.args.ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				sort.Strings(tt.want.DeviceIds)
				sort.Strings(got.DeviceIds)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
