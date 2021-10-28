package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/platform"
	"github.com/plgd-dev/go-coap/v2/message"
	caService "github.com/plgd-dev/hub/certificate-authority/test"
	coapgwTest "github.com/plgd-dev/hub/coap-gateway/test"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	grpcgwService "github.com/plgd-dev/hub/grpc-gateway/test"
	httpgwTest "github.com/plgd-dev/hub/http-gateway/test"
	"github.com/plgd-dev/hub/http-gateway/uri"
	idService "github.com/plgd-dev/hub/identity-store/test"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	raService "github.com/plgd-dev/hub/resource-aggregate/test"
	rdService "github.com/plgd-dev/hub/resource-directory/test"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	"github.com/plgd-dev/hub/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/test/pb"
	testService "github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandler_GetPendingMetadataUpdates(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		accept         string
		deviceIdFilter string
	}
	tests := []struct {
		name string
		args args
		want []*pb.PendingCommand
	}{
		{
			name: "get pending metadata updates",
			args: args{
				accept:         uri.ApplicationProtoJsonContentType,
				deviceIdFilter: deviceID,
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_DeviceMetadataUpdatePending{
						DeviceMetadataUpdatePending: &events.DeviceMetadataUpdatePending{
							DeviceId: deviceID,
							UpdatePending: &events.DeviceMetadataUpdatePending_ShadowSynchronization{
								ShadowSynchronization: commands.ShadowSynchronization_DISABLED,
							},
							AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	testService.ClearDB(ctx, t)
	oauthShutdown := oauthTest.SetUp(t)
	idShutdown := idService.SetUp(t)
	raShutdown := raService.SetUp(t)
	rdShutdown := rdService.SetUp(t)
	grpcShutdown := grpcgwService.SetUp(t)
	caShutdown := caService.SetUp(t)
	secureGWShutdown := coapgwTest.SetUp(t)

	defer caShutdown()
	defer grpcShutdown()
	defer rdShutdown()
	defer raShutdown()
	defer idShutdown()
	defer oauthShutdown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	secureGWShutdown()

	create := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.CreateResource(ctx, &pb.CreateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
		})
		require.Error(t, err)
	}
	create()
	retrieve := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.GetResourceFromDevice(ctx, &pb.GetResourceFromDeviceRequest{
			ResourceId: commands.NewResourceID(deviceID, platform.ResourceURI),
		})
		require.Error(t, err)
	}
	retrieve()
	update := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.UpdateResource(ctx, &pb.UpdateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
		})
		require.Error(t, err)
	}
	update()
	delete := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.DeleteResource(ctx, &pb.DeleteResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
		})
		require.Error(t, err)
	}
	delete()
	updateDeviceMetadata := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
			DeviceId:              deviceID,
			ShadowSynchronization: pb.UpdateDeviceMetadataRequest_DISABLED,
		})
		require.Error(t, err)
	}
	updateDeviceMetadata()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodGet, uri.AliasDevicePendingMetadataUpdates, nil).AuthToken(token).Accept(tt.args.accept)
			rb.DeviceId(tt.args.deviceIdFilter)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()

			var values []*pb.PendingCommand

			for {
				var v pb.PendingCommand
				err = Unmarshal(resp.StatusCode, resp.Body, &v)
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
				values = append(values, &v)
			}
			pbTest.CmpPendingCmds(t, tt.want, values)
		})
	}
}
