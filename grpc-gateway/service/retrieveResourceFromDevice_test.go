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
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message"
)

func TestRequestHandler_GetResourceFromDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req pb.GetResourceFromDeviceRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *events.ResourceRetrieved
		wantErr bool
	}{
		{
			name: "valid /light/2",
			args: args{
				req: pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/light/2"),
				},
			},
			want: &events.ResourceRetrieved{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     "/light/2",
				},
				Content: &commands.Content{
					CoapContentFormat: int32(message.AppOcfCbor),
					ContentType:       message.AppOcfCbor.String(),
				},
				Status: commands.Status_OK,
			},
		},
		{
			name: "valid /oic/d",
			args: args{
				req: pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
				},
			},
			want: &events.ResourceRetrieved{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     "/oic/d",
				},
				Content: &commands.Content{
					CoapContentFormat: int32(message.AppOcfCbor),
					ContentType:       message.AppOcfCbor.String(),
				},
				Status: commands.Status_OK,
			},
		},
		{
			name: "invalid Href",
			args: args{
				req: pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/unknown"),
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
			got, err := c.GetResourceFromDevice(ctx, &tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, got.GetData())
				got.GetData().EventMetadata = nil
				got.GetData().AuditContext = nil
				got.GetData().Content.Data = nil
				test.CheckProtobufs(t, tt.want, got.GetData(), test.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
