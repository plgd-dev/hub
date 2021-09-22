package service

import (
	"context"
	"sort"
	"testing"

	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/stretchr/testify/require"
)

const jwtWithSubTestUser = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0VXNlcklEIn0.6EZJidMCJ5UMwyttpwUNer-GdsBmPH1_ckH8ZU-SRpo`
const jwtWithSubTestUser2 = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0VXNlcjIifQ.LfeMsf6VObU3BcT0PmsO_ryDd_V2B712gBdlKed_2no`

func TestService_DeleteDevices(t *testing.T) {
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
			name: "invalid deviceId",
			args: args{
				ctx:     grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserId),
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
				ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubAaa),
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
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubTestUser),
			},
			want: &pb.DeleteDevicesResponse{DeviceIds: []string{testDevID1}},
		},
		{
			name: "multiple",
			args: args{
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubTestUser),
				request: &pb.DeleteDevicesRequest{
					DeviceIds: []string{testDevID2, testDevID3},
				},
			},
			want: &pb.DeleteDevicesResponse{DeviceIds: []string{testDevID2, testDevID3}},
		},
		{
			name: "duplicit",
			args: args{
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubTestUser),
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
				ctx: kitNetGrpc.CtxWithIncomingToken(context.Background(), jwtWithSubTestUser),
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
