package client_test

import (
	"context"
	"errors"
	"testing"
	"time"

	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_CreateResource(t *testing.T) {
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
	}
	tests := []struct {
		name        string
		args        args
		want        interface{}
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "create resource",
			args: args{
				token:    oauthTest.GetServiceToken(t),
				deviceID: deviceID,
				href:     "/oic/d",
				data: map[string]interface{}{
					"n": "devsim - valid update value",
				},
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
				data: map[string]interface{}{
					"n": "devsim",
				},
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
			err := c.CreateResource(ctx, tt.args.deviceID, tt.args.href, tt.args.data, &got)
			if tt.wantErr {
				require.Error(t, err)
				var grpcStatus interface {
					GRPCStatus() *status.Status
				}
				errors.As(err, &grpcStatus)
				assert.Equal(t, tt.wantErrCode.String(), grpcStatus.GRPCStatus().Code().String())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
