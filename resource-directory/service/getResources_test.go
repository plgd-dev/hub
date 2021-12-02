package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"

	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/test/pb"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandler_GetResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.GetResourcesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.Resource
	}{
		{
			name: "valid",
			args: args{
				req: &pb.GetResourcesRequest{
					ResourceIdFilter: []string{
						commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")).ToString(),
					},
				},
			},
			want: []*pb.Resource{
				{
					Types: []string{types.CORE_LIGHT},
					Data: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceLightInstanceHref("1"), "",
						map[string]interface{}{
							"state": false,
							"power": uint64(0),
							"name":  "Light",
						},
					),
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetResources(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			values := make([]*pb.Resource, 0, 1)
			for {
				value, err := client.Recv()
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
				values = append(values, value)
			}
			pbTest.CmpResourceValues(t, tt.want, values)
		})
	}
}
