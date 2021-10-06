package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/client"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"github.com/plgd-dev/cloud/v2/test"
	testCfg "github.com/plgd-dev/cloud/v2/test/config"
	oauthTest "github.com/plgd-dev/cloud/v2/test/oauth-server/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_UpdateResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	type args struct {
		token    string
		deviceID string
		href     string
		data     interface{}
		opts     []client.UpdateOption
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "valid - update value",
			args: args{
				token:    oauthTest.GetDefaultServiceToken(t),
				deviceID: deviceID,
				href:     "/oc/con",
				data: map[string]interface{}{
					"n": "devsim - valid update value",
				},
			},
			want: map[interface{}]interface{}{
				"n": "devsim - valid update value",
			},
		},
		{
			name: "valid - revert update",
			args: args{
				token:    oauthTest.GetDefaultServiceToken(t),
				deviceID: deviceID,
				href:     "/oc/con",
				data: map[string]interface{}{
					"n": test.TestDeviceName,
				},
			},
			want: map[interface{}]interface{}{
				"n": test.TestDeviceName,
			},
		},
		{
			name: "valid with resourceInterface",
			args: args{
				token:    oauthTest.GetDefaultServiceToken(t),
				deviceID: deviceID,
				href:     "/oc/con",
				data: map[string]interface{}{
					"n": test.TestDeviceName,
				},
				opts: []client.UpdateOption{client.WithInterface("oic.if.baseline")},
			},
			want: map[interface{}]interface{}{
				"n": test.TestDeviceName,
			},
		},
		{
			name: "invalid href",
			args: args{
				token:    oauthTest.GetDefaultServiceToken(t),
				deviceID: deviceID,
				href:     "/invalid/href",
				data: map[string]interface{}{
					"n": "devsim",
				},
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
			err := c.UpdateResource(ctx, tt.args.deviceID, tt.args.href, tt.args.data, &got, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
