package service_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/device/schema/platform"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	idService "github.com/plgd-dev/hub/v2/identity-store/test"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/service"
	"github.com/plgd-dev/hub/v2/resource-aggregate/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func TestPublishUnpublish(t *testing.T) {
	config := test.MakeConfig(t)
	config.APIs.GRPC.Addr = "localhost:9888"
	config.Clients.Eventstore.SnapshotThreshold = 1

	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()

	idShutdown := idService.SetUp(t)
	defer idShutdown()
	cfg := log.MakeDefaultConfig()
	cfg.Level = zap.DebugLevel
	log.Setup(cfg)
	raShutdown := test.New(t, config)
	defer raShutdown()

	ctx := kitNetGrpc.CtxWithToken(context.Background(), oauthTest.GetDefaultAccessToken(t))

	idConn, err := client.New(testCfg.MakeGrpcClientConfig(config.Clients.IdentityStore.Connection.Addr), log.Get(), trace.NewNoopTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = idConn.Close()
	}()
	idClient := pbIS.NewIdentityStoreClient(idConn.GRPC())

	raConn, err := client.New(testCfg.MakeGrpcClientConfig(config.APIs.GRPC.Addr), log.Get(), trace.NewNoopTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	raClient := service.NewResourceAggregateClient(raConn.GRPC())

	deviceID := "dev0"
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

	cfg.Level = zap.DebugLevel
	log.Setup(cfg)
	pubReq := testMakePublishResourceRequest(deviceID, []string{href})
	_, err = raClient.PublishResourceLinks(ctx, pubReq)
	require.NoError(t, err)

	unpubReq := testMakeUnpublishResourceRequest(deviceID, []string{href})
	_, err = raClient.UnpublishResourceLinks(ctx, unpubReq)
	require.NoError(t, err)
}
