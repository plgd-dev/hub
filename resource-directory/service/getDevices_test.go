package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	isTest "github.com/plgd-dev/hub/v2/identity-store/test"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	virtualdevice "github.com/plgd-dev/hub/v2/test/virtual-device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerGetDevicesParallel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	numDevices := 100
	numResources := 1
	numParallelRequests := 10

	isConfig := isTest.MakeConfig(t)
	isConfig.APIs.GRPC.TLS.ClientCertificateRequired = false

	raConfig := raTest.MakeConfig(t)
	raConfig.APIs.GRPC.TLS.ClientCertificateRequired = false

	tearDown := service.SetUp(ctx, t, service.WithISConfig(isConfig), service.WithRAConfig(raConfig))
	defer tearDown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))
	virtualdevice.CreateDevices(ctx, t, numDevices, numResources, test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	getDevices := func(c pb.GrpcGatewayClient) {
		client, err := c.GetDevices(ctx, &pb.GetDevicesRequest{})
		require.NoError(t, err)
		numCurr := 0
		for {
			dev, err := client.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
			require.NotEmpty(t, dev)
			numCurr++
		}
		require.Equal(t, numDevices, numCurr)
	}

	getDevices(c)

	var wg sync.WaitGroup
	wg.Add(numParallelRequests)
	t.Logf("starting %v requests\n", numParallelRequests)
	for i := range numParallelRequests {
		go func(v int) {
			defer wg.Done()
			n := time.Now()
			getDevices(c)
			t.Logf("%v getDevices client %v\n", v, -1*time.Until(n))
		}(i)
	}
	wg.Wait()
}

func TestRequestHandlerGetDevices(t *testing.T) {
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
					Types:       []string{types.DEVICE_CLOUD, device.ResourceType},
					Interfaces:  []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
					Id:          deviceID,
					Name:        test.TestDeviceName,
					ModelNumber: test.TestDeviceModelNumber,
					Metadata: &pb.Device_Metadata{
						Connection: &commands.Connection{
							Status:   commands.Connection_ONLINE,
							Protocol: test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
						},
						TwinEnabled: true,
						TwinSynchronization: &commands.TwinSynchronization{
							State: commands.TwinSynchronization_IN_SYNC,
						},
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

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
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
					if errors.Is(err, io.EOF) {
						break
					}
					require.NoError(t, err)
					assert.NotEmpty(t, dev.GetProtocolIndependentId())
					assert.NotEmpty(t, dev.GetData().GetContent().GetData())
					dev.ProtocolIndependentId = ""
					dev.Metadata.Connection.Id = ""
					dev.Metadata.Connection.ConnectedAt = 0
					dev.Metadata.Connection.LocalEndpoints = nil
					dev.Metadata.Connection.ServiceId = ""
					dev.Metadata.TwinSynchronization.SyncingAt = 0
					dev.Metadata.TwinSynchronization.InSyncAt = 0
					dev.Metadata.TwinSynchronization.CommandMetadata = nil
					dev.Data = nil
					devices = append(devices, dev)
				}
				test.CheckProtobufs(t, tt.want, devices, test.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
