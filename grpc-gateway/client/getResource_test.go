package client_test

import (
	"context"
	"testing"
	"time"

	kitNetGrpc "github.com/go-ocf/kit/net/grpc"

	authTest "github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/grpc-gateway/client"
	grpcTest "github.com/go-ocf/cloud/grpc-gateway/test"
	"github.com/stretchr/testify/require"
)

func TestClient_GetResource(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
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
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/oc/con",
			},
			want: map[interface{}]interface{}{
				"n": grpcTest.TestDeviceName,
			},
		},
		{
			name: "valid with skip shadow",
			args: args{
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/oc/con",
				opts:     []client.GetOption{client.WithSkipShadow()},
			},
			want: map[interface{}]interface{}{
				"n": grpcTest.TestDeviceName,
			},
		},
		{
			name: "valid with interface",
			args: args{
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/oc/con",
				opts:     []client.GetOption{client.WithInterface("oic.if.baseline")},
			},
			wantErr: false,
			want: map[interface{}]interface{}{
				"n":  grpcTest.TestDeviceName,
				"if": []interface{}{"oic.if.rw", "oic.if.baseline"},
				"rt": []interface{}{"oic.wk.con"},
			},
		},
		{
			name: "valid with interface and skip shadow",
			args: args{
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/oc/con",
				opts:     []client.GetOption{client.WithSkipShadow(), client.WithInterface("oic.if.baseline")},
			},
			wantErr: false,
			want: map[interface{}]interface{}{
				"n":  grpcTest.TestDeviceName,
				"if": []interface{}{"oic.if.rw", "oic.if.baseline"},
				"rt": []interface{}{"oic.wk.con"},
			},
		},
		{
			name: "invalid href",
			args: args{
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/invalid/href",
			},
			wantErr: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)

	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()

	c := NewTestClient(t)
	defer c.Close(context.Background())

	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())
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
