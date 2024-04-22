package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
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
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resourceLinks := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, resourceLinks)
	defer shutdownDevSim()
	const switchID = "1"
	resourceLinks = append(resourceLinks, test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)...)
	// for update resource-directory cache
	time.Sleep(time.Second)

	// get resource from device via HUB
	lightResourceData, err := c.GetResourceFromDevice(ctx, &pb.GetResourceFromDeviceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
	})
	require.NoError(t, err)

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
					ResourceIdFilter: []*pb.ResourceIdFilter{
						{
							ResourceId: commands.NewResourceID("unknown", ""),
						},
					},
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
					ResourceIdFilter: []*pb.ResourceIdFilter{
						{
							ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
						},
					},
				},
			},
			want: []*pb.Resource{
				{
					Types: []string{types.CORE_LIGHT},
					Data: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "",
						map[string]interface{}{
							"state": false,
							"power": uint64(0),
							"name":  "Light",
						}),
				},
			},
		},
		{
			name: "valid resourceIdFilter with ETag",
			args: args{
				req: &pb.GetResourcesRequest{
					ResourceIdFilter: []*pb.ResourceIdFilter{
						{
							ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
							Etag: [][]byte{
								lightResourceData.GetData().GetEtag(),
							},
						},
					},
				},
			},
			want: []*pb.Resource{
				{
					Types: []string{types.CORE_LIGHT},
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: deviceID,
							Href:     test.TestResourceLightInstanceHref("1"),
						},
						Status: commands.Status_NOT_MODIFIED,
						Content: &commands.Content{
							CoapContentFormat: -1,
						},
						AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
						ResourceTypes: test.TestResourceLightInstanceResourceTypes,
					},
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
					Data: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesInstanceHref(switchID), test.TestResourceSwitchesInstanceResourceTypes, "",
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
				if errors.Is(err, io.EOF) {
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
