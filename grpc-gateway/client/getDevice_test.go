package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	test "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

const (
	TestTimeout               = time.Second * 20
	DeviceSimulatorIdNotFound = "00000000-0000-0000-0000-000000000111"
)

func NewTestDeviceSimulator(deviceID, deviceName string, withResources bool) client.DeviceDetails {
	var resources []*commands.Resource
	if withResources {
		resources = append(resources, test.ResourceLinksToResources(deviceID, test.GetAllBackendResourceLinks())...)
		resources = test.SortResources(resources)
	}

	return client.DeviceDetails{
		ID: deviceID,
		Device: &pb.Device{
			Id:          deviceID,
			Name:        deviceName,
			ModelNumber: test.TestDeviceModelNumber,
			Types:       []string{types.DEVICE_CLOUD, device.ResourceType},
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
			Interfaces:      []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
			OwnershipStatus: pb.Device_OWNED,
		},
		Resources: resources,
	}
}

func TestClient_GetDevice(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		token    string
		deviceID string
	}
	tests := []struct {
		name    string
		args    args
		want    client.DeviceDetails
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
			},
			want: NewTestDeviceSimulator(deviceID, test.TestDeviceName, true),
		},
		{
			name: "not-found",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: "not-found",
			},
			wantErr: true,
		},
	}

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	c := grpcgwTest.NewTestClient(t)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			got, err := c.GetDevice(runctx, tt.args.deviceID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got.Resources = test.CleanUpResourcesArray(got.Resources)
			require.NotEmpty(t, got.Device.GetProtocolIndependentId())
			got.Device.ProtocolIndependentId = ""
			got.Device.Metadata.Connection.Id = ""
			got.Device.Metadata.Connection.ConnectedAt = 0
			got.Device.Metadata.Connection.LocalEndpoints = nil
			got.Device.Metadata.Connection.ServiceId = ""
			got.Device.Metadata.TwinSynchronization.SyncingAt = 0
			got.Device.Metadata.TwinSynchronization.InSyncAt = 0
			got.Device.Metadata.TwinSynchronization.CommandMetadata = nil
			require.NotEmpty(t, got.Device.GetData().GetContent().GetData())
			got.Device.Data = nil
			test.CheckProtobufs(t, tt.want, got, test.RequireToCheckFunc(require.Equal))
		})
	}
}
