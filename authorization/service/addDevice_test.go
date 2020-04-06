package service

import (
	"context"
	"testing"

	"github.com/go-ocf/ocf-cloud/authorization/pb"
	"github.com/stretchr/testify/require"
)

func TestService_AddDevice(t *testing.T) {
	type args struct {
		ctx     context.Context
		request *pb.AddDeviceRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.AddDeviceResponse
		wantErr bool
	}{
		{
			name: "invalid userId",
			args: args{
				ctx:     context.Background(),
				request: &pb.AddDeviceRequest{},
			},
			wantErr: true,
		},
		{
			name: "invalid deviceId",
			args: args{
				ctx: context.Background(),
				request: &pb.AddDeviceRequest{
					UserId: "userId",
				},
			},
			wantErr: true,
		},
		{
			name: "not belongs to user",
			args: args{
				request: &pb.AddDeviceRequest{
					DeviceId: testDeviceID,
					UserId:   "aaa",
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				request: &pb.AddDeviceRequest{
					UserId:   "userId",
					DeviceId: "deviceId",
				},
			},
			want: &pb.AddDeviceResponse{},
		},
		{
			name: "duplicit",
			args: args{
				request: &pb.AddDeviceRequest{
					UserId:   "userId",
					DeviceId: "deviceId",
				},
			},
			want: &pb.AddDeviceResponse{},
		},
		// TODO: Add test cases.
	}

	s, shutdown := newTestService(t)
	defer shutdown()
	defer s.cleanUp()
	persistDevice(t, s.service.persistence, newTestDevice())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.service.AddDevice(context.Background(), tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
