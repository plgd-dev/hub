package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/grpc-gateway/client"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	test "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
)

func sortDevices(s map[string]*client.DeviceDetails) map[string]*client.DeviceDetails {
	for key, x := range s {
		x.Resources = test.CleanUpResourcesArray(x.Resources)
		x.Device.ProtocolIndependentId = ""
		x.Device.Metadata.Status.ValidUntil = 0
		s[key] = x
	}

	return s
}

func TestClient_GetDevices(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	tearDown := test.SetUp(ctx, t)
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
				token: oauthTest.GetServiceToken(t),
			},
			want: map[string]client.DeviceDetails{
				deviceID: NewTestDeviceSimulator(deviceID, test.TestDeviceName, false),
			},
		},
		{
			name: "not-found",
			args: args{
				token: oauthTest.GetServiceToken(t),
				opts:  []client.GetDevicesOption{client.WithResourceTypes("not-found")},
			},
			wantErr: false,
		},
	}

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	c := NewTestClient(t)
	defer c.Close(context.Background())

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			got, err := c.GetDevices(ctx, tt.args.opts...)
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
