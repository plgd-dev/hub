package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/hub/grpc-gateway/client"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	test "github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/stretchr/testify/assert"
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
				token: oauthTest.GetDefaultServiceToken(t),
			},
			want: map[string]client.DeviceDetails{
				deviceID: NewTestDeviceSimulator(deviceID, test.TestDeviceName, false),
			},
		},
		{
			name: "not-found",
			args: args{
				token: oauthTest.GetDefaultServiceToken(t),
				opts:  []client.GetDevicesOption{client.WithResourceTypes("not-found")},
			},
			wantErr: false,
		},
	}

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	c := NewTestClient(t)
	defer func() {
		err := c.Close(context.Background())
		assert.NoError(t, err)
	}()

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
