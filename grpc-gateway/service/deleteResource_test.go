package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	exCodes "github.com/plgd-dev/hub/grpc-gateway/pb/codes"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/test/pb"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func TestRequestHandlerDeleteResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		href string
		ttl  int64
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "/light/1 - MethodNotAllowed",
			args: args{
				href: test.TestResourceLightInstanceHref("1"),
			},
			wantErr:     true,
			wantErrCode: codes.Code(exCodes.MethodNotAllowed),
		},
		{
			name: "invalid Href",
			args: args{
				href: "/unknown",
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
		{
			name: "/oic/d - PermissionDenied",
			args: args{
				href: device.ResourceURI,
			},
			wantErr:     true,
			wantErrCode: codes.PermissionDenied,
		},
		{
			name: "invalid timeToLive",
			args: args{
				href: test.TestResourceLightInstanceHref("1"),
				ttl:  int64(99 * time.Millisecond),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "not found - delete /switches/-1",
			args: args{
				href: test.TestResourceSwitchesInstanceHref("-1"),
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
		{
			name: "delete /switches/1",
			args: args{
				href: test.TestResourceSwitchesInstanceHref("1"),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, "1", "2", "3")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.DeleteResourceRequest{
				ResourceId: commands.NewResourceID(deviceID, tt.args.href),
				TimeToLive: tt.args.ttl,
			}
			got, err := c.DeleteResource(ctx, req)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrCode, status.Convert(err).Code())
				return
			}
			require.NoError(t, err)

			want := pbTest.MakeResourceDeleted(t, deviceID, tt.args.href)
			pbTest.CmpResourceDeleted(t, want, got.GetData())
		})
	}
}
