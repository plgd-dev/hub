package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/plgd-dev/go-coap/v2/message"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

func cmpResourceValues(t *testing.T, want []*pb.ResourceValue, got []*pb.ResourceValue) {
	require.Len(t, got, len(want))
	for idx := range want {
		dataWant := want[idx].GetContent().GetData()
		datagot := got[idx].GetContent().GetData()
		want[idx].Content.Data = nil
		got[idx].Content.Data = nil
		test.CheckProtobufs(t, want[idx], got[idx], test.RequireToCheckFunc(require.Equal))
		w := test.DecodeCbor(t, dataWant)
		g := test.DecodeCbor(t, datagot)
		require.Equal(t, w, g)
	}
}

func TestRequestHandler_RetrieveResourcesValues(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.RetrieveResourcesValuesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.ResourceValue
	}{
		{
			name: "valid",
			args: args{
				req: &pb.RetrieveResourcesValuesRequest{
					ResourceIdsFilter: []*commands.ResourceId{
						{
							DeviceId: deviceID,
							Href:     "/light/1",
						},
					},
				},
			},
			want: []*pb.ResourceValue{
				{
					ResourceId: &commands.ResourceId{
						DeviceId: deviceID,
						Href:     "/light/1",
					},
					Types: []string{"core.light"},
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"state": false,
							"power": uint64(0),
							"name":  "Light",
						}),
					},
					Status: pb.Status_OK,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.RetrieveResourcesValues(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				values := make([]*pb.ResourceValue, 0, 1)
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
