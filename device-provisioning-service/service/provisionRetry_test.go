package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
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

type testDpsHandlerConfig struct {
	failLimit      uint64
	failCode       codes.Code
	expectedCounts map[test.HandlerID]uint64
}

func defaultTestDpsHandlerConfig() testDpsHandlerConfig {
	return testDpsHandlerConfig{
		failLimit: 0,
		failCode:  codes.ServiceUnavailable,
		expectedCounts: map[test.HandlerID]uint64{
			test.HandlerIDDefault:            0,
			test.HandlerIDTime:               1,
			test.HandlerIDOwnership:          1,
			test.HandlerIDCredentials:        1,
			test.HandlerIDACLs:               1,
			test.HandlerIDCloudConfiguration: 1,
		},
	}
}

func newTestRequestHandler(t *testing.T, dpsCfg service.Config, handlerCfg testDpsHandlerConfig) *test.RequestHandlerWithCounter {
	return test.NewRequestHandlerWithCounter(t, dpsCfg, func(h test.HandlerID, count uint64) codes.Code {
		switch h {
		case test.HandlerIDTime, test.HandlerIDOwnership, test.HandlerIDCredentials, test.HandlerIDACLs, test.HandlerIDCloudConfiguration:
			if count < handlerCfg.failLimit {
				return handlerCfg.failCode
			}
		case test.HandlerIDDefault:
		}
		return 0
	}, func(defaultHandlerCount, processTimeCount, processOwnershipCount, processCloudConfigurationCount, processCredentialsCount, processACLsCount uint64,
	) (bool, error) {
		expDefaultHandlerCount := handlerCfg.expectedCounts[test.HandlerIDDefault]
		expTimeCount := handlerCfg.expectedCounts[test.HandlerIDTime]
		expOwnershipCount := handlerCfg.expectedCounts[test.HandlerIDOwnership]
		expCloudConfigurationCount := handlerCfg.expectedCounts[test.HandlerIDCloudConfiguration]
		expCredentialsCount := handlerCfg.expectedCounts[test.HandlerIDCredentials]
		expACLsCount := handlerCfg.expectedCounts[test.HandlerIDACLs]
		if defaultHandlerCount > expDefaultHandlerCount ||
			processTimeCount > expTimeCount ||
			processOwnershipCount > expOwnershipCount ||
			processCloudConfigurationCount > expCloudConfigurationCount ||
			processCredentialsCount > expCredentialsCount ||
			processACLsCount > expACLsCount {
			return false, fmt.Errorf("invalid counters: defaultHandlerCounter=(%v:%v) processTimeCount=(%v:%v) processOwnershipCounter=(%v:%v) processCloudConfigurationCounter=(%v:%v) processCredentialsCounter=(%v:%v) processACLsCounter=(%v:%v)",
				defaultHandlerCount, expDefaultHandlerCount,
				processTimeCount, expTimeCount,
				processOwnershipCount, expOwnershipCount,
				processCloudConfigurationCount, expCloudConfigurationCount,
				processCredentialsCount, expCredentialsCount,
				processACLsCount, expACLsCount)
		}
		return defaultHandlerCount == expDefaultHandlerCount &&
			processTimeCount == expTimeCount &&
			processOwnershipCount == expOwnershipCount &&
			processCloudConfigurationCount == expCloudConfigurationCount &&
			processCredentialsCount == expCredentialsCount &&
			processACLsCount == expACLsCount, nil
	})
}

func TestForceReprovisioning(t *testing.T) {
	dpsCfg := test.MakeConfig(t)
	rh := newTestRequestHandler(t, dpsCfg, defaultTestDpsHandlerConfig())
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|
		hubTestService.SetUpServicesId|hubTestService.SetUpServicesCoapGateway|hubTestService.SetUpServicesResourceAggregate|hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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
	dpsShutDown := test.New(t, rh.Cfg())
	deferedDpsCleanUp := true
	defer func() {
		if deferedDpsCleanUp {
			dpsShutDown()
		}
	}()
	deviceID, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, rh.Cfg().APIs.COAP.Addr, test.TestDevsimResources)
	defer shutdownSim()
	deferedDpsCleanUp = false
	dpsShutDown()

	subClient, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)
	defer func() {
		errC := subClient.CloseSend()
		require.NoError(t, errC)
	}()

	err = test.ForceReprovision(ctx, c, deviceID)
	require.NoError(t, err)

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

	rh.StartDps(service.WithRequestHandler(rh))
	defer rh.StopDps()

	err = rh.Verify(ctx)
	require.NoError(t, err)

	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_OFFLINE)
	require.NoError(t, err)
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_ONLINE)
	require.NoError(t, err)
}

func TestProvisioningTransientFailureRetry(t *testing.T) {
	const transientFailLimit = uint64(3)
	// the testing device is configured to force full reprovisioning after 3 transient failures of a single step,
	// after the first 3 failures of a step it will then always succeeds, so full reprovisioning will be forced 3 times,
	// resulting in the following counts
	const expectedTimeCount = transientFailLimit + 5
	const expectedOwnershipCount = transientFailLimit + 4
	const expectedCloudConfigurationCount = transientFailLimit + 3
	const expectedCredentialsCount = transientFailLimit + 2
	const expectedACLsCount = transientFailLimit + 1
	rh := test.NewRequestHandlerWithExpectedCounters(t, transientFailLimit, codes.ServiceUnavailable, expectedTimeCount, expectedOwnershipCount, expectedCloudConfigurationCount, expectedCredentialsCount, expectedACLsCount)
	testProvisioningWithDPSHandler(t, rh, 5*time.Minute)
}

func TestProvisioningRetry(t *testing.T) {
	const failLimit = uint64(1)
	// non transient failure forces full reprovisioning, even previously successful steps are retried
	const expectedTimeCount = failLimit + 5
	const expectedOwnershipCount = failLimit + 4
	const expectedCloudConfigurationCount = failLimit + 3
	const expectedCredentialsCount = failLimit + 2
	const expectedACLsCount = failLimit + 1
	rh := test.NewRequestHandlerWithExpectedCounters(t, failLimit, codes.InternalServerError, expectedTimeCount, expectedOwnershipCount, expectedCloudConfigurationCount, expectedCredentialsCount, expectedACLsCount)
	testProvisioningWithDPSHandler(t, rh, 3*time.Minute)
}
