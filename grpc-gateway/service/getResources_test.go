package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/test/pb"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerGetResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resourceLinks := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, resourceLinks)
	defer shutdownDevSim()
	const switchId = "1"
	resourceLinks = append(resourceLinks, test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchId)...)
	time.Sleep(200 * time.Millisecond)

	type args struct {
		req *pb.GetResourcesRequest
	}
	tests := []struct {
		name  string
		args  args
		cmpFn func(*testing.T, []*pb.Resource, []*pb.Resource)
		want  []*pb.Resource
	}{
		{
			name: "invalid deviceIdFilter",
			args: args{
				req: &pb.GetResourcesRequest{
					DeviceIdFilter: []string{"unknown"},
				},
			},
		},
		{
			name: "invalid resourceIdFilter",
			args: args{
				req: &pb.GetResourcesRequest{
					ResourceIdFilter: []string{"unknown"},
				},
			},
		},
		{
			name: "invalid typeFilter",
			args: args{
				req: &pb.GetResourcesRequest{
					TypeFilter: []string{"unknown"},
				},
			},
		},
		{
			name: "valid deviceIdFilter",
			args: args{
				req: &pb.GetResourcesRequest{
					DeviceIdFilter: []string{deviceID},
				},
			},
			cmpFn: pbTest.CmpResourceValuesBasic,
			want:  test.ResourceLinksToResources2(deviceID, resourceLinks),
		},
		{
			name: "valid resourceIdFilter",
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
						}),
				},
			},
		},
		{
			name: "valid typeFilter",
			args: args{
				req: &pb.GetResourcesRequest{
					TypeFilter: []string{types.BINARY_SWITCH},
				},
			},
			want: []*pb.Resource{
				{
					Types: []string{types.BINARY_SWITCH},
					Data: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesInstanceHref(switchId), "",
						map[string]interface{}{
							"value": false,
						}),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetResources(ctx, tt.args.req)
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
			if tt.cmpFn != nil {
				tt.cmpFn(t, tt.want, values)
				return
			}
			pbTest.CmpResourceValues(t, tt.want, values)
		})
	}
}
