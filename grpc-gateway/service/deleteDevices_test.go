package service_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerDeleteDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.DeleteDevicesRequest
	}
	tests := []struct {
		name string
		args args
		want *pb.DeleteDevicesResponse
	}{
		{
			name: "not owned device",
			args: args{
				req: &pb.DeleteDevicesRequest{
					DeviceIdFilter: []string{"badId"},
				},
			},
			want: &pb.DeleteDevicesResponse{
				DeviceIds: nil,
			},
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

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := c.DeleteDevices(ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.DeviceIds, resp.DeviceIds)
		})
	}
}
