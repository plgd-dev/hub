package service_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandler_DeleteDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.DeleteDevicesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *pb.DeleteDevicesResponse
	}{
		{
			name: "invalid deviceID",
			args: args{
				req: &pb.DeleteDevicesRequest{
					DeviceIdFilter: []string{"badId"},
				},
			},
			wantErr: true,
		},
		{
			name: "all owned devices",
			args: args{
				req: &pb.DeleteDevicesRequest{
					DeviceIdFilter: []string{},
				},
			},
			want: &pb.DeleteDevicesResponse{
				DeviceIds: []string{deviceID},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := c.DeleteDevices(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				require.Equal(t, tt.want.DeviceIds, resp.DeviceIds)
			}
		})
	}
}
