package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message/codes"
	caService "github.com/plgd-dev/hub/v2/certificate-authority/test"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// A provisioned device stores data into storage, so when the device process is restarted
// no reprovisioning should be done. The data should be loaded from storage and only cloud
// events should be republished.
func TestReprovisioningAfterRestart(t *testing.T) {
	const failLimit = uint64(0)
	const expectedTimeCount = failLimit + 1
	const expectedOwnershipCount = failLimit + 1
	const expectedCloudConfigurationCount = failLimit + 1
	const expectedCredentialsCount = failLimit + 1
	const expectedACLsCount = failLimit + 1
	h := test.NewRequestHandlerWithExpectedCounters(t, failLimit, codes.InternalServerError, expectedTimeCount, expectedOwnershipCount,
		expectedCloudConfigurationCount, expectedCredentialsCount, expectedACLsCount)

	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|
		hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId|hubTestService.SetUpServicesCoapGateway|hubTestService.SetUpServicesResourceAggregate|
		hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	ctx = pkgGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	caCfg := caService.MakeConfig(t)
	caCfg.Signer.ExpiresIn = time.Hour * 10
	caShutdown := caService.New(t, caCfg)
	defer caShutdown()

	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	dpsShutDown := test.New(t, h.Cfg())
	deferedDpsCleanUp := true
	defer func() {
		if deferedDpsCleanUp {
			dpsShutDown()
		}
	}()
	deviceID, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, h.Cfg().APIs.COAP.Addr, test.TestDevsimResources)
	defer shutdownSim()
	deferedDpsCleanUp = false
	dpsShutDown()

	err = test.ForceReprovision(ctx, c, deviceID)
	require.NoError(t, err)

	subClient, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)
	defer func() {
		errC := subClient.CloseSend()
		require.NoError(t, errC)
	}()
	subID, corID := test.SubscribeToEvents(t, subClient, &pb.SubscribeToEvents{
		CorrelationId: "deviceOnline",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
				},
			},
		},
	})

	h.StartDps(service.WithRequestHandler(h))
	defer h.StopDps()

	err = h.Verify(ctx)
	require.NoError(t, err)

	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_OFFLINE)
	require.NoError(t, err)
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_ONLINE)
	require.NoError(t, err)

	h.Logf("restarting device")
	err = test.RestartDockerContainer(test.TestDockerObtContainerName)
	require.NoError(t, err)

	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_OFFLINE)
	require.NoError(t, err)
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_ONLINE)
	require.NoError(t, err)

	err = h.Verify(ctx)
	require.NoError(t, err)
}
