package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
)

func cmpResourceValues(t *testing.T, want []*pb.Resource, got []*pb.Resource) {
	require.Len(t, got, len(want))
	for idx := range want {
		dataWant := want[idx].Data.GetContent().GetData()
		datagot := got[idx].Data.GetContent().GetData()
		w1 := &pb.Resource{
			Types: want[idx].GetTypes(),
			Data: &events.ResourceChanged{
				ResourceId: want[idx].Data.GetResourceId(),
				Content: &commands.Content{
					ContentType: want[idx].Data.GetContent().GetContentType(),
				},
				Status: want[idx].Data.GetStatus(),
			},
		}
		w2 := &pb.Resource{
			Types: got[idx].GetTypes(),
			Data: &events.ResourceChanged{
				ResourceId: got[idx].Data.GetResourceId(),
				Content: &commands.Content{
					ContentType: got[idx].Data.GetContent().GetContentType(),
				},
				Status: got[idx].Data.GetStatus(),
			},
		}
		w2.Data.Content.Data = nil
		test.CheckProtobufs(t, w1, w2, test.RequireToCheckFunc(require.Equal))
		w := test.DecodeCbor(t, dataWant)
		g := test.DecodeCbor(t, datagot)
		require.Equal(t, w, g)
	}
}

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
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: deviceID,
							Href:     test.TestResourceLightInstanceHref("1"),
						},
						Content: &commands.Content{
							ContentType: message.AppOcfCbor.String(),
							Data: test.EncodeToCbor(t, map[string]interface{}{
								"state": false,
								"power": uint64(0),
								"name":  "Light",
								"if":    []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
								"rt":    []interface{}{types.CORE_LIGHT},
							}),
						},
						Status: commands.Status_OK,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetResources(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
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
				cmpResourceValues(t, tt.want, values)
			}
		})
	}
}
