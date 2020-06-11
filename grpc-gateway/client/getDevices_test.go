package client_test

import (
	"context"
	"testing"
	"time"

	authTest "github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/grpc-gateway/client"
	test "github.com/go-ocf/cloud/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/stretchr/testify/require"
)

func sortDevices(s map[string]client.DeviceDetails) map[string]client.DeviceDetails {
	for key, x := range s {
		x.Resources = test.SortResources(x.Resources)
		s[key] = x
	}

	return s
}

func TestClient_GetDevices(t *testing.T) {
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
				token: authTest.UserToken,
			},
			want: map[string]client.DeviceDetails{
				deviceID: NewTestDeviceSimulator(deviceID, test.TestDeviceName),
			},
		},
		{
			name: "not-found",
			args: args{
				token: authTest.UserToken,
				opts:  []client.GetDevicesOption{client.WithResourceTypes("not-found")},
			},
			wantErr: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	c := NewTestClient(t)
	defer c.Close(context.Background())

	shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
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
			require.Equal(t, tt.want, got)
		})
	}
}
