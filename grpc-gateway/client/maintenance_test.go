package client_test

import (
	"context"
	"testing"
	"time"

	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	test "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
)

const RebootTakes = time.Second * 8 // for reboot
const RebootTimeout = TestTimeout + RebootTakes

func TestClient_FactoryReset(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	type args struct {
		deviceID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "factory reset - maintenance resource is not published",
			args: args{
				deviceID: deviceID,
			},
			wantErr: true,
		},
		{
			name: "not found",
			args: args{
				deviceID: "notFound",
			},
			wantErr: true,
		},
	}

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	c := NewTestClient(t)
	defer c.Close(context.Background())

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.FactoryReset(ctx, tt.args.deviceID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_Reboot(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), RebootTimeout)
	defer cancel()
	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	type args struct {
		deviceID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "reboot - maintenance resource is not published",
			args: args{
				deviceID: deviceID,
			},
			wantErr: true,
		},
		{
			name: "not found",
			args: args{
				deviceID: "notFound",
			},
			wantErr: true,
		},
	}

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	c := NewTestClient(t)
	defer c.Close(context.Background())

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Reboot(ctx, tt.args.deviceID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				time.Sleep(RebootTakes)
			}
		})
	}
}
