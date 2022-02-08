package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"

	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandler_GetDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.GetDevicesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.Device
	}{
		{
			name: "valid",
			args: args{
				req: &pb.GetDevicesRequest{},
			},
			want: []*pb.Device{
				{
					Types:      []string{types.DEVICE_CLOUD, device.ResourceType},
					Interfaces: []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
					Id:         deviceID,
					Name:       test.TestDeviceName,
					Metadata: &pb.Device_Metadata{
						Status: &commands.ConnectionStatus{
							Value: commands.ConnectionStatus_ONLINE,
						},
					},
				},
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
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetDevices(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				devices := make([]*pb.Device, 0, 1)
				for {
					dev, err := client.Recv()
					if err == io.EOF {
						break
					}
					require.NoError(t, err)
					assert.NotEmpty(t, dev.ProtocolIndependentId)
					dev.ProtocolIndependentId = ""
					dev.Metadata.Status.ValidUntil = 0
					devices = append(devices, dev)
				}
				test.CheckProtobufs(t, tt.want, devices, test.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
