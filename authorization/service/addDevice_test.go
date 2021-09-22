package service

import (
	"context"
	"testing"

	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/stretchr/testify/require"
)

const jwtWithSubUserId = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VySWQifQ.sK7h3M0UhwXqc_vgkjl9MKIR41me7Np2-YUIHOijcSA`
const jwtWithSubAaa = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhYWEifQ.QDa8-bP8MvjX8I2QQMYVVQ5utSMRMdgHOVoE2hUWlos`

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
				ctx:     grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserId),
				request: &pb.AddDeviceRequest{},
			},
			wantErr: true,
		},
		{
			name: "not belongs to user",
			args: args{
				ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubAaa),
				request: &pb.AddDeviceRequest{
					DeviceId: testDeviceID,
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserId),
				request: &pb.AddDeviceRequest{
					DeviceId: "deviceId",
				},
			},
			want: &pb.AddDeviceResponse{},
		},
		{
			name: "duplicit",
			args: args{
				ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserId),
				request: &pb.AddDeviceRequest{
					DeviceId: "deviceId",
				},
			},
			want: &pb.AddDeviceResponse{},
		},
		// TODO: Add test cases.
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
			got, err := s.service.AddDevice(tt.args.ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
