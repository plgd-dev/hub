package service_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message"
)

func TestRequestHandler_UpdateResourcesValues(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req pb.UpdateResourceRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.UpdateResourceResponse
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				req: pb.UpdateResourceRequest{
					ResourceId: &commands.ResourceId{
						DeviceId: deviceID,
						Href:     "/light/1",
					},
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 1,
						}),
					},
				},
			},
			want: &pb.UpdateResourceResponse{
				Content: &pb.Content{},
				Status:  pb.Status_OK,
			},
		},
		{
			name: "valid with interface",
			args: args{
				req: pb.UpdateResourceRequest{
					ResourceInterface: "oic.if.baseline",
					ResourceId: &commands.ResourceId{
						DeviceId: deviceID,
						Href:     "/light/1",
					},
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 2,
						}),
					},
				},
			},
			want: &pb.UpdateResourceResponse{
				Content: &pb.Content{},
				Status:  pb.Status_OK,
			},
		},
		{
			name: "revert update",
			args: args{
				req: pb.UpdateResourceRequest{
					ResourceInterface: "oic.if.baseline",
					ResourceId: &commands.ResourceId{
						DeviceId: deviceID,
						Href:     "/light/1",
					},
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 0,
						}),
					},
				},
			},
			want: &pb.UpdateResourceResponse{
				Content: &pb.Content{},
				Status:  pb.Status_OK,
			},
		},
		{
			name: "update RO-resource",
			args: args{
				req: pb.UpdateResourceRequest{
					ResourceId: &commands.ResourceId{
						DeviceId: deviceID,
						Href:     "/oic/d",
					},
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"di": "abc",
						}),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid Href",
			args: args{
				req: pb.UpdateResourceRequest{
					ResourceId: &commands.ResourceId{
						DeviceId: deviceID,
						Href:     "/unknown",
					},
				},
			},
			wantErr: true,
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

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.UpdateResource(ctx, &tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			test.CheckProtobufs(t, tt.want, got, test.RequireToCheckFunc(require.Equal))
		})
	}
}
