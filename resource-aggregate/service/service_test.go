package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/plgd-dev/device/v2/schema/platform"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestPublishUnpublish(t *testing.T) {
	cfg := raTest.MakeConfig(t)
	cfg.APIs.GRPC.Addr = "localhost:9888"

	fmt.Println("cfg: ", cfg)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	const services = hubTestService.SetUpServicesMachine2MachineOAuth | hubTestService.SetUpServicesOAuth | hubTestService.SetUpServicesId | hubTestService.SetUpServicesResourceAggregate
	tearDown := hubTestService.SetUpServices(ctx, t, services, hubTestService.WithRAConfig(cfg))
	defer tearDown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	idConn, err := client.New(config.MakeGrpcClientConfig(cfg.Clients.IdentityStore.Connection.Addr), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = idConn.Close()
	}()
	idClient := pbIS.NewIdentityStoreClient(idConn.GRPC())

	raConn, err := client.New(config.MakeGrpcClientConfig(cfg.APIs.GRPC.Addr), fileWatcher, log.Get(), noop.NewTracerProvider())
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

	pubReq := testMakePublishResourceRequest(deviceID, []string{href})
	_, err = raClient.PublishResourceLinks(ctx, pubReq)
	require.NoError(t, err)

	unpubReq := testMakeUnpublishResourceRequest(deviceID, []string{href})
	_, err = raClient.UnpublishResourceLinks(ctx, unpubReq)
	require.NoError(t, err)
}
