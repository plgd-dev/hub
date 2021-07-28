package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message"
)

func cmpResourceValues(t *testing.T, want []*pb.Resource, got []*pb.Resource) {
	require.Len(t, got, len(want))
	for idx := range want {
		dataWant := want[idx].Data.GetContent().GetData()
		datagot := got[idx].Data.GetContent().GetData()
		want[idx].Data.Content.Data = nil
		got[idx].Data.Content.Data = nil
		test.CheckProtobufs(t, want[idx], got[idx], test.RequireToCheckFunc(require.Equal))
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
						commands.NewResourceID(deviceID, "/light/1").ToString(),
					},
				},
			},
			want: []*pb.Resource{
				{
					Types: []string{"core.light"},
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: deviceID,
							Href:     "/light/1",
						},
						Content: &commands.Content{
							CoapContentFormat: int32(message.AppOcfCbor),
							ContentType:       message.AppOcfCbor.String(),
							Data: test.EncodeToCbor(t, map[string]interface{}{
								"state": false,
								"power": uint64(0),
								"name":  "Light",
								"if":    []interface{}{"oic.if.rw", "oic.if.baseline"},
								"rt":    []interface{}{"core.light"},
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
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

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
					value.Data.AuditContext = nil
					value.Data.EventMetadata = nil
					values = append(values, value)
				}
				cmpResourceValues(t, tt.want, values)
			}
		})
	}
}
