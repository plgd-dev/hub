package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/go-coap/v3/message"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerGetPendingMetadataUpdates(t *testing.T) {
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
				accept:         pkgHttp.ApplicationProtoJsonContentType,
				deviceIdFilter: deviceID,
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_DeviceMetadataUpdatePending{
						DeviceMetadataUpdatePending: &events.DeviceMetadataUpdatePending{
							DeviceId: deviceID,
							UpdatePending: &events.DeviceMetadataUpdatePending_TwinEnabled{
								TwinEnabled: false,
							},
							AuditContext: commands.NewAuditContext(service.DeviceUserID, "", service.DeviceUserID),
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	testService.ClearDB(ctx, t)
	var closeFunc fn.FuncList
	defer closeFunc.Execute()
	tearDown := testService.SetUpServices(ctx, t, testService.SetUpServicesOAuth|testService.SetUpServicesMachine2MachineOAuth|testService.SetUpServicesId|testService.SetUpServicesResourceAggregate|
		testService.SetUpServicesResourceDirectory|testService.SetUpServicesCertificateAuthority|testService.SetUpServicesGrpcGateway)
	closeFunc.AddFunc(tearDown)

	deferedSecureGWShutdown := true
	secureGWShutdown := coapgwTest.SetUp(t)
	defer func() {
		if deferedSecureGWShutdown {
			secureGWShutdown()
		}
	}()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	deferedSecureGWShutdown = false
	secureGWShutdown()

	createFn := func() {
		createCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, errC := c.CreateResource(createCtx, &pb.CreateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
		})
		require.Error(t, errC)
	}
	createFn()
	retrieveFn := func() {
		retrieveCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, errG := c.GetResourceFromDevice(retrieveCtx, &pb.GetResourceFromDeviceRequest{
			ResourceId: commands.NewResourceID(deviceID, platform.ResourceURI),
		})
		require.Error(t, errG)
	}
	retrieveFn()
	updateFn := func() {
		updateCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, errU := c.UpdateResource(updateCtx, &pb.UpdateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
		})
		require.Error(t, errU)
	}
	updateFn()
	deleteFn := func() {
		deleteCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, errD := c.DeleteResource(deleteCtx, &pb.DeleteResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
		})
		require.Error(t, errD)
	}
	deleteFn()
	updateDeviceMetadataFn := func() {
		updateCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, errU := c.UpdateDeviceMetadata(updateCtx, &pb.UpdateDeviceMetadataRequest{
			DeviceId:    deviceID,
			TwinEnabled: false,
		})
		require.Error(t, errU)
	}
	updateDeviceMetadataFn()

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
				err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &v)
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				values = append(values, &v)
			}
			pbTest.CmpPendingCmds(t, tt.want, values)
		})
	}
}
