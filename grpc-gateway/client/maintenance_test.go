package client_test

import (
	"context"
	"testing"
	"time"

	authTest "github.com/plgd-dev/cloud/authorization/provider"
	test "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/stretchr/testify/require"
)

const RebootTakes = time.Second * 8 // for reboot
const RebootTimeout = TestTimeout + RebootTakes

func TestClient_FactoryReset(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
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

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	c := NewTestClient(t)
	defer c.Close(context.Background())

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
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

	ctx, cancel := context.WithTimeout(context.Background(), RebootTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	c := NewTestClient(t)
	defer c.Close(context.Background())

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
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
