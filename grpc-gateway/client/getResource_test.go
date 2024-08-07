package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestClientGetResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	type args struct {
		token    string
		deviceID string
		href     string
		opts     []client.GetOption
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     configuration.ResourceURI,
			},
			want: map[interface{}]interface{}{
				"n": test.TestDeviceName,
			},
		},
		{
			name: "valid with skip twin",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				opts:     []client.GetOption{client.WithSkipTwin()},
			},
			want: map[interface{}]interface{}{
				"n": test.TestDeviceName,
			},
		},
		{
			name: "valid with interface",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				opts:     []client.GetOption{client.WithInterface(interfaces.OC_IF_BASELINE)},
			},
			wantErr: false,
			want: map[interface{}]interface{}{
				"n":  test.TestDeviceName,
				"if": []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
				"rt": []interface{}{configuration.ResourceType},
			},
		},
		{
			name: "valid with interface and skip twin",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				opts:     []client.GetOption{client.WithSkipTwin(), client.WithInterface(interfaces.OC_IF_BASELINE)},
			},
			wantErr: false,
			want: map[interface{}]interface{}{
				"n":  test.TestDeviceName,
				"if": []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
				"rt": []interface{}{configuration.ResourceType},
			},
		},
		{
			name: "invalid href",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     "/invalid/href",
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
			var got interface{}
			err := c.GetResource(runctx, tt.args.deviceID, tt.args.href, &got, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
