package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/plgd-dev/device/v2/schema/platform"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	idService "github.com/plgd-dev/hub/v2/identity-store/test"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestPublishUnpublish(t *testing.T) {
	cfg := raTest.MakeConfig(t)
	cfg.APIs.GRPC.Addr = "localhost:9888"

	fmt.Println("cfg: ", cfg)

	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()

	idShutdown := idService.SetUp(t)
	defer idShutdown()
	logCfg := log.MakeDefaultConfig()
	logCfg.Level = log.DebugLevel
	log.Setup(logCfg)
	raShutdown := raTest.New(t, cfg)
	defer raShutdown()

	ctx := kitNetGrpc.CtxWithToken(context.Background(), oauthTest.GetDefaultAccessToken(t))

	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	idConn, err := client.New(ctx, config.MakeGrpcClientConfig(cfg.Clients.IdentityStore.Connection.Addr), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = idConn.Close()
	}()
	idClient := pbIS.NewIdentityStoreClient(idConn.GRPC())

	raConn, err := client.New(ctx, config.MakeGrpcClientConfig(cfg.APIs.GRPC.Addr), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	raClient := service.NewResourceAggregateClient(raConn.GRPC())

	deviceID := test.GenerateDeviceIDbyIdx(0)
	href := platform.ResourceURI
	_, err = idClient.AddDevice(ctx, &pbIS.AddDeviceRequest{
		DeviceId: deviceID,
	})
	require.NoError(t, err)
	defer func() {
		_, err = idClient.DeleteDevices(ctx, &pbIS.DeleteDevicesRequest{
			DeviceIds: []string{deviceID},
		})
		require.NoError(t, err)
	}()

	logCfg.Level = log.DebugLevel
	log.Setup(logCfg)
	pubReq := testMakePublishResourceRequest(deviceID, []string{href})
	_, err = raClient.PublishResourceLinks(ctx, pubReq)
	require.NoError(t, err)

	unpubReq := testMakeUnpublishResourceRequest(deviceID, []string{href})
	_, err = raClient.UnpublishResourceLinks(ctx, unpubReq)
	require.NoError(t, err)
}
