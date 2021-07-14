package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/grpc-gateway/client"
	extCodes "github.com/plgd-dev/cloud/grpc-gateway/pb/codes"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"google.golang.org/grpc/codes"

	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
)

func TestClient_DeleteResource(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
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
				token:    oauthTest.GetServiceToken(t),
				deviceID: deviceID,
				href:     "/ligh/1",
			},
			wantErr:     true,
			wantErrCode: codes.Code(extCodes.MethodNotAllowed),
		},
		{
			name: "/ligh/1 - permission denied",
			args: args{
				token:    oauthTest.GetServiceToken(t),
				deviceID: deviceID,
				href:     "/oic/d",
			},
			wantErr:     true,
			wantErrCode: codes.PermissionDenied,
		},
		{
			name: "invalid href",
			args: args{
				token:    oauthTest.GetServiceToken(t),
				deviceID: deviceID,
				href:     "/invalid/href",
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
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
