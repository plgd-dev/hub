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
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	test "github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
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
			Id:    deviceID,
			Name:  deviceName,
			Types: []string{types.DEVICE_CLOUD, device.ResourceType},
			Metadata: &pb.Device_Metadata{
				Status: &commands.ConnectionStatus{
					Value: commands.ConnectionStatus_ONLINE,
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

	c := NewTestClient(t)
	defer func() {
		err := c.Close(context.Background())
		assert.NoError(t, err)
	}()

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.COAPS_TCP_SCHEME+testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			got, err := c.GetDevice(ctx, tt.args.deviceID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got.Resources = test.CleanUpResourcesArray(got.Resources)
			require.NotEmpty(t, got.Device.GetProtocolIndependentId())
			got.Device.ProtocolIndependentId = ""
			got.Device.Metadata.Status.ValidUntil = 0
			got.Device.Metadata.Status.ConnectionId = ""
			require.NotEmpty(t, got.Device.GetData().GetContent().GetData())
			got.Device.Data = nil
			test.CheckProtobufs(t, tt.want, got, test.RequireToCheckFunc(require.Equal))
		})
	}
}
