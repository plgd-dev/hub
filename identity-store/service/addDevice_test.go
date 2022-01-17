package service

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestServiceAddDevice(t *testing.T) {
	jwtWithSubAaa := config.CreateJwtToken(t, jwt.MapClaims{
		"sub": "aaa",
	})
	jwtWithSubUserID := config.CreateJwtToken(t, jwt.MapClaims{
		"sub": "userId",
	})
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
				ctx:     grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
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
				ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
				request: &pb.AddDeviceRequest{
					DeviceId: "deviceId",
				},
			},
			want: &pb.AddDeviceResponse{},
		},
		{
			name: "duplicit",
			args: args{
				ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
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
