package client_test

import (
	"context"
	"testing"
	"time"

	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"

	"github.com/plgd-dev/hub/grpc-gateway/client"
	"github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	tearDown := test.SetUp(ctx, t)
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
				token:    oauthTest.GetDefaultServiceToken(t),
				deviceID: deviceID,
				href:     "/oc/con",
			},
			want: map[interface{}]interface{}{
				"n":  test.TestDeviceName,
				"if": []interface{}{"oic.if.rw", "oic.if.baseline"},
				"rt": []interface{}{"oic.wk.con"},
			},
		},
		{
			name: "valid with skip shadow",
			args: args{
				token:    oauthTest.GetDefaultServiceToken(t),
				deviceID: deviceID,
				href:     "/oc/con",
				opts:     []client.GetOption{client.WithSkipShadow()},
			},
			want: map[interface{}]interface{}{
				"n": test.TestDeviceName,
			},
		},
		{
			name: "valid with interface",
			args: args{
				token:    oauthTest.GetDefaultServiceToken(t),
				deviceID: deviceID,
				href:     "/oc/con",
				opts:     []client.GetOption{client.WithInterface("oic.if.baseline")},
			},
			wantErr: false,
			want: map[interface{}]interface{}{
				"n":  test.TestDeviceName,
				"if": []interface{}{"oic.if.rw", "oic.if.baseline"},
				"rt": []interface{}{"oic.wk.con"},
			},
		},
		{
			name: "valid with interface and skip shadow",
			args: args{
				token:    oauthTest.GetDefaultServiceToken(t),
				deviceID: deviceID,
				href:     "/oc/con",
				opts:     []client.GetOption{client.WithSkipShadow(), client.WithInterface("oic.if.baseline")},
			},
			wantErr: false,
			want: map[interface{}]interface{}{
				"n":  test.TestDeviceName,
				"if": []interface{}{"oic.if.rw", "oic.if.baseline"},
				"rt": []interface{}{"oic.wk.con"},
			},
		},
		{
			name: "invalid href",
			args: args{
				token:    oauthTest.GetDefaultServiceToken(t),
				deviceID: deviceID,
				href:     "/invalid/href",
			},
			wantErr: true,
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
			var got interface{}
			err := c.GetResource(ctx, tt.args.deviceID, tt.args.href, &got, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
