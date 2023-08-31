package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerGetDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.GetDevicesRequest
	}
	tests := []struct {
		name string
		args args
		want []*pb.Device
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
						Connection: &commands.Connection{
							Status:   commands.Connection_ONLINE,
							Protocol: test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
							Service:  &commands.Connection_Service{},
						},
						TwinSynchronization: &commands.TwinSynchronization{
							State: commands.TwinSynchronization_IN_SYNC,
						},
						TwinEnabled: true,
					},
					OwnershipStatus: pb.Device_OWNED,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	// for update resource-directory cache
	time.Sleep(time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetDevices(ctx, tt.args.req)
			require.NoError(t, err)
			devices := make([]*pb.Device, 0, 1)
			for {
				dev, err := client.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				assert.NotEmpty(t, dev.ProtocolIndependentId)
				assert.NotEmpty(t, dev.GetData().GetContent().GetData())
				assert.NotEmpty(t, dev.GetMetadata().GetConnection().GetService().GetId())
				devices = append(devices, dev)
			}
			pbTest.CmpDeviceValues(t, tt.want, devices)
		})
	}
}
