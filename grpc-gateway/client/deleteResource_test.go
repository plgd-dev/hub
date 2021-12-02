package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/hub/grpc-gateway/client"
	extCodes "github.com/plgd-dev/hub/grpc-gateway/pb/codes"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestClient_DeleteResource(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		token    string
		deviceID string
		href     string
		opts     []client.DeleteOption
	}
	tests := []struct {
		name        string
		args        args
		want        interface{}
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "/ligh/1 - method not allowd",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     "/ligh/1",
			},
			wantErr:     true,
			wantErrCode: codes.Code(extCodes.MethodNotAllowed),
		},
		{
			name: "/ligh/1 - permission denied",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     device.ResourceURI,
			},
			wantErr:     true,
			wantErrCode: codes.PermissionDenied,
		},
		{
			name: "invalid href",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     "/invalid/href",
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
	}

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

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
			err := c.DeleteResource(ctx, tt.args.deviceID, tt.args.href, &got, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
