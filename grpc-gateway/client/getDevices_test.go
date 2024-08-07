package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	test "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func sortDevices(s map[string]*client.DeviceDetails) map[string]*client.DeviceDetails {
	for key, x := range s {
		x.Resources = test.CleanUpResourcesArray(x.Resources)
		x.Device.ProtocolIndependentId = ""
		x.Device.Metadata.Connection.Id = ""
		x.Device.Metadata.Connection.ConnectedAt = 0
		x.Device.Metadata.Connection.LocalEndpoints = nil
		x.Device.Metadata.Connection.ServiceId = ""
		x.Device.Metadata.TwinSynchronization.SyncingAt = 0
		x.Device.Metadata.TwinSynchronization.InSyncAt = 0
		x.Device.Metadata.TwinSynchronization.CommandMetadata = nil
		x.Device.Data = nil
		s[key] = x
	}

	return s
}

func TestClient_GetDevices(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		token string
		opts  []client.GetDevicesOption
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]client.DeviceDetails
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				token: oauthTest.GetDefaultAccessToken(t),
			},
			want: map[string]client.DeviceDetails{
				deviceID: NewTestDeviceSimulator(deviceID, test.TestDeviceName, false),
			},
		},
		{
			name: "not-found",
			args: args{
				token: oauthTest.GetDefaultAccessToken(t),
				opts:  []client.GetDevicesOption{client.WithResourceTypes("not-found")},
			},
			wantErr: false,
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
			got, err := c.GetDevices(runctx, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got = sortDevices(got)
			test.CheckProtobufs(t, tt.want, got, test.RequireToCheckFunc(require.Equal))
		})
	}
}
