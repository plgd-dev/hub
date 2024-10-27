package service_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	caService "github.com/plgd-dev/hub/v2/certificate-authority/test"
	hubCoapGWTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	grpcPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const caCfgSignerValidFrom = "now-1h"

type testRequestHandlerWithExpiringCert struct {
	service.RequestHandle

	blocking  atomic.Bool   // block provisioning
	expiresIn time.Duration // wait this long on the final call (ProcessCloudConfiguration) so the certificate expires
	expired   atomic.Bool
	logger    log.Logger
	wait      chan struct{}
	waiting   atomic.Bool
}

func newTestRequestHandlerWithExpiredCert(expiresIn time.Duration) *testRequestHandlerWithExpiringCert {
	return &testRequestHandlerWithExpiringCert{
		expiresIn: expiresIn,
		logger:    log.NewLogger(log.Config{Level: zapcore.DebugLevel}),
		wait:      make(chan struct{}),
	}
}

func (h *testRequestHandlerWithExpiringCert) blockProvisioning() {
	if h.blocking.CompareAndSwap(false, true) {
		h.logger.Debugf("start blocking provisioning")
	}
}

func (h *testRequestHandlerWithExpiringCert) unblockProvisioning() {
	if h.blocking.CompareAndSwap(true, false) {
		h.logger.Debugf("stop blocking provisioning")
	}
}

func (h *testRequestHandlerWithExpiringCert) ProcessACLs(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if h.expiresIn == 0 {
		h.expired.Store(true)
		return h.RequestHandle.ProcessACLs(ctx, req, session, linkedHubs, group)
	}
	if h.blocking.Load() {
		// returning nil, nil will result in the resp being discarded and nothing being sent back to the client
		return nil, nil //nolint:nilnil
	}
	if h.waiting.CompareAndSwap(false, true) {
		h.blockProvisioning() // block retry of provisioning/certificate refresh until we manually unblock
		wait := h.expiresIn + time.Second
		time.Sleep(wait)
		h.logger.Debugf("certificate expired")
		h.expired.Store(true)
	}
	h.logger.Debugf("process acls")
	return h.RequestHandle.ProcessACLs(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithExpiringCert) verify(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("provisioning timed out: expired=%v", h.expired.Load())
		case <-time.After(time.Second):
			h.logger.Debugf("expired=%v", h.expired.Load())
		}
		if h.expired.Load() {
			return nil
		}
	}
}

func TestProvisioningWithExpiringCertificate(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|
		hubTestService.SetUpServicesId|hubTestService.SetUpServicesResourceAggregate|hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	coapGWCfg := hubCoapGWTest.MakeConfig(t)
	coapGWCfg.APIs.COAP.TLS.Embedded.ClientCertificateRequired = true
	coapGWCfg.APIs.COAP.TLS.DisconnectOnExpiredCertificate = true
	coapGWCfg.APIs.COAP.OwnerCacheExpiration = time.Second
	coapGWShutdown := hubCoapGWTest.New(t, coapGWCfg)
	deferedCoapGWCleanUp := true
	defer func() {
		if deferedCoapGWCleanUp {
			coapGWShutdown()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := grpcPb.NewGrpcGatewayClient(conn)

	caCfg := caService.MakeConfig(t)
	caShutdown := caService.New(t, caCfg)
	deferedCaCleanUp := true
	defer func() {
		if deferedCaCleanUp {
			caShutdown()
		}
	}()

	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	dpsCfg := test.MakeConfig(t)
	dpsCfg.APIs.COAP.InactivityMonitor.Timeout = time.Minute
	rh := newTestRequestHandler(t, dpsCfg, defaultTestDpsHandlerConfig())
	rh.StartDps(service.WithRequestHandler(rh))
	deferedDpsCleanUp := true
	defer func() {
		if deferedDpsCleanUp {
			rh.StopDps()
		}
	}()
	deviceID, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, dpsCfg.APIs.COAP.Addr, test.TestDevsimResources)
	defer shutdownSim()

	// wait for provisioning success
	err = rh.Verify(ctx)
	require.NoError(t, err)
	deferedDpsCleanUp = false
	rh.StopDps()

	shortTimeout := time.Second * 30 // enough time for provisioning to succeed and certificate to expire
	shortCtx, shortCancel := context.WithTimeout(context.Background(), shortTimeout)
	defer shortCancel()
	shortCtx = pkgGrpc.CtxWithToken(shortCtx, token)

	subClient, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(shortCtx)
	require.NoError(t, err)
	defer func(s grpcPb.GrpcGateway_SubscribeToEventsClient) {
		errC := s.CloseSend()
		require.NoError(t, errC)
	}(subClient)

	subID, corID := test.SubscribeToEvents(t, subClient, &grpcPb.SubscribeToEvents{
		CorrelationId: "deviceOnline",
		Action: &grpcPb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &grpcPb.SubscribeToEvents_CreateSubscription{
				EventFilter: []grpcPb.SubscribeToEvents_CreateSubscription_Event{
					grpcPb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
				},
			},
		},
	})

	err = test.ForceReprovision(ctx, c, deviceID)
	require.NoError(t, err)

	// shutdown coap-gw to force device to reconnect
	deferedCoapGWCleanUp = false
	coapGWShutdown()

	// shutdown coap-gw sets device to offline, wait for it
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_OFFLINE)
	require.NoError(t, err)

	// reconfigure CA to use certificate that will expire soon
	deferedCaCleanUp = false
	caShutdown()
	// sign with certificate that will soon expire
	expiresIn := 13 * time.Second // 10s + 3s, where 10s = expiring limit of the test device, 3s is enough time to finish provisioning steps
	caCfg.Signer.ValidFrom = caCfgSignerValidFrom
	caCfg.Signer.ExpiresIn = time.Hour + expiresIn
	caShutdown = caService.New(t, caCfg)
	deferedCaCleanUp = true

	h := newTestRequestHandlerWithExpiredCert(expiresIn)
	dpsShutDown := test.New(t, dpsCfg, service.WithRequestHandler(h))
	defer dpsShutDown()

	// DPS provisioning should succeed
	err = h.verify(shortCtx)
	require.NoError(t, err)

	coapGWShutdown = hubCoapGWTest.New(t, coapGWCfg)
	defer coapGWShutdown()

	// online msg should not be received because of the expired certificate -> wait for timeout
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_ONLINE)
	require.Error(t, err)

	h.unblockProvisioning()

	subClient, err = client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)
	defer func(s grpcPb.GrpcGateway_SubscribeToEventsClient) {
		errC := s.CloseSend()
		require.NoError(t, errC)
	}(subClient)
	subID, corID = test.SubscribeToEvents(t, subClient, &grpcPb.SubscribeToEvents{
		CorrelationId: "deviceOnline",
		Action: &grpcPb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &grpcPb.SubscribeToEvents_CreateSubscription{
				EventFilter: []grpcPb.SubscribeToEvents_CreateSubscription_Event{
					grpcPb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
				},
			},
		},
	})

	deferedCaCleanUp = false
	caShutdown()
	// sign with valid certificate
	caCfg.Signer.ExpiresIn = time.Hour * 2
	caShutdown = caService.New(t, caCfg)
	defer caShutdown()

	// online event -> after successful reprovisioning
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_ONLINE)
	require.NoError(t, err)
}

func TestProvisioningWithExpiredCertificate(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|
		hubTestService.SetUpServicesId|hubTestService.SetUpServicesResourceAggregate|hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	coapGWCfg := hubCoapGWTest.MakeConfig(t)
	coapGWCfg.APIs.COAP.TLS.Embedded.ClientCertificateRequired = true
	coapGWShutdown := hubCoapGWTest.New(t, coapGWCfg)
	defer coapGWShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := grpcPb.NewGrpcGatewayClient(conn)

	caCfg := caService.MakeConfig(t)
	caShutdown := caService.New(t, caCfg)
	deferedCaCleanUp := true
	defer func() {
		if deferedCaCleanUp {
			caShutdown()
		}
	}()

	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	dpsCfg := test.MakeConfig(t)
	rh := newTestRequestHandler(t, dpsCfg, defaultTestDpsHandlerConfig())
	rh.StartDps(service.WithRequestHandler(rh))
	deferedDpsCleanUp := true
	defer func() {
		if deferedDpsCleanUp {
			rh.StopDps()
		}
	}()
	deviceID, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, dpsCfg.APIs.COAP.Addr, test.TestDevsimResources)
	defer shutdownSim()

	// wait for provisioning success so we get OFFLINE event first after reprovisioning
	err = rh.Verify(ctx)
	require.NoError(t, err)
	deferedDpsCleanUp = false
	rh.StopDps()

	err = test.ForceReprovision(ctx, c, deviceID)
	require.NoError(t, err)

	deferedCaCleanUp = false
	caShutdown()
	// sign with expired certificate
	caCfg.Signer.ValidFrom = caCfgSignerValidFrom
	caCfg.Signer.ExpiresIn = time.Second
	caShutdown = caService.New(t, caCfg)
	defer caShutdown()

	const expectedSuccessCount = 1
	h := test.NewRequestHandlerWithCounter(t, dpsCfg, nil,
		func(defaultHandlerCount, processTimeCount, processOwnershipCount, processCloudConfigurationCount, processCredentialsCount, processACLsCount uint64) (bool, error) {
			if defaultHandlerCount > 0 ||
				processTimeCount > expectedSuccessCount ||
				processOwnershipCount > expectedSuccessCount ||
				processCloudConfigurationCount > expectedSuccessCount ||
				processCredentialsCount > expectedSuccessCount ||
				processACLsCount > expectedSuccessCount {
				return false, fmt.Errorf("invalid counters default(%d:%d) time(%d:%d) owner(%d:%d) cloud(%d:%d) creds(%d:%d) acls(%d:%d)",
					defaultHandlerCount, 0,
					processTimeCount, expectedSuccessCount,
					processOwnershipCount, expectedSuccessCount,
					processCloudConfigurationCount, expectedSuccessCount,
					processCredentialsCount, expectedSuccessCount,
					processACLsCount, expectedSuccessCount,
				)
			}
			return false, nil
		})

	h.StartDps(service.WithRequestHandler(h))
	defer h.StopDps()

	// DPS provisioning should fail and reprovisioning should be triggered
	shortCtx, shortCancel := context.WithTimeout(context.Background(), time.Second*20)
	defer shortCancel()
	shortCtx = pkgGrpc.CtxWithToken(shortCtx, token)
	err = h.Verify(shortCtx)
	require.Error(t, err)
}

type checkAuthCountsFn func(verifyCertificateCount, verifyConnectionCount uint64) bool

type testRequestHandlerWithAuthCounter struct {
	test.RequestHandlerWithDps
	a                        service.AuthHandler
	verifyCertificateCounter atomic.Uint64
	verifyConnectionCounter  atomic.Uint64
	checkAuthCounts          checkAuthCountsFn
	checkFinalAuthCounts     checkAuthCountsFn
	service.RequestHandle
}

func newTestRequestHandlerWithAuthCounter(t *testing.T, dpsCfg service.Config, egCache *service.EnrollmentGroupsCache, checkAuthCounts, checkFinalAuthCounts checkAuthCountsFn) *testRequestHandlerWithAuthCounter {
	return &testRequestHandlerWithAuthCounter{
		RequestHandlerWithDps: test.MakeRequestHandlerWithDps(t, dpsCfg),
		a:                     service.MakeDefaultAuthHandler(dpsCfg, egCache),
		checkAuthCounts:       checkAuthCounts,
		checkFinalAuthCounts:  checkFinalAuthCounts,
	}
}

func (h *testRequestHandlerWithAuthCounter) GetChainsCache() *cache.Cache[uint64, [][]*x509.Certificate] {
	return h.a.GetChainsCache()
}

func (h *testRequestHandlerWithAuthCounter) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	h.verifyCertificateCounter.Inc()
	return h.a.VerifyPeerCertificate(rawCerts, verifiedChains)
}

func (h *testRequestHandlerWithAuthCounter) VerifyConnection(cs tls.ConnectionState) error {
	h.verifyConnectionCounter.Inc()
	return h.a.VerifyConnection(cs)
}

func (h *testRequestHandlerWithAuthCounter) verify(ctx context.Context) error {
	logCounter := 0
	for h.checkAuthCounts(h.verifyCertificateCounter.Load(), h.verifyConnectionCounter.Load()) {
		select {
		case <-ctx.Done():
			return errors.New("verification timed out")
		case <-time.After(time.Second):
			logCounter++
			if logCounter%3 == 0 {
				h.Logf("verifyCertificateCounter=%v verifyConnectionCounter=%v",
					h.verifyCertificateCounter.Load(),
					h.verifyConnectionCounter.Load())
			}
		}
	}

	h.Logf("final verifyCertificateCounter=%v verifyConnectionCounter=%v",
		h.verifyCertificateCounter.Load(),
		h.verifyConnectionCounter.Load())
	if !h.checkFinalAuthCounts(h.verifyCertificateCounter.Load(), h.verifyConnectionCounter.Load()) {
		return fmt.Errorf("unexpected counters verifyCertificateCounter=%v verifyConnectionCounter=%v",
			h.verifyCertificateCounter.Load(),
			h.verifyConnectionCounter.Load())
	}
	return nil
}

func TestProvisioningWithDeletedEnrollmentGroup(t *testing.T) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesCertificateAuthority|hubTestService.SetUpServicesMachine2MachineOAuth|
		hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId|hubTestService.SetUpServicesCoapGateway|hubTestService.SetUpServicesResourceAggregate|
		hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := grpcPb.NewGrpcGatewayClient(conn)

	deviceID := hubTest.MustFindDeviceByName(test.TestDeviceObtName)
	dpsCfg := test.MakeConfig(t)
	dpsShutDown := test.New(t, dpsCfg)
	deferedDpsCleanUp := true
	defer func() {
		if deferedDpsCleanUp {
			dpsShutDown()
		}
	}()
	deviceID, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, dpsCfg.APIs.COAP.Addr, test.TestDevsimResources)
	defer shutdownSim()
	deferedDpsCleanUp = false
	dpsShutDown()

	subClient, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)
	defer func(s grpcPb.GrpcGateway_SubscribeToEventsClient) {
		errC := s.CloseSend()
		require.NoError(t, errC)
	}(subClient)

	subID, corID := test.SubscribeToEvents(t, subClient, &grpcPb.SubscribeToEvents{
		CorrelationId: "deviceOnline",
		Action: &grpcPb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &grpcPb.SubscribeToEvents_CreateSubscription{
				EventFilter: []grpcPb.SubscribeToEvents_CreateSubscription_Event{
					grpcPb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
				},
			},
		},
	})

	err = test.ForceReprovision(ctx, c, deviceID)
	require.NoError(t, err)

	dpsCfg.EnrollmentGroups = nil
	store, storeTearDown := test.NewMongoStore(t)
	defer storeTearDown()
	count, err := store.DeleteEnrollmentGroups(ctx, test.DPSOwner, &pb.GetEnrollmentGroupsRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	count, err = store.DeleteHubs(ctx, test.DPSOwner, &pb.GetHubsRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	ah := newTestRequestHandlerWithAuthCounter(t, dpsCfg, service.NewEnrollmentGroupsCache(ctx, time.Minute, store, log.Get()), func(verifyCertificateCount, _ uint64) bool {
		// verification of certificates was tried multiple times
		return verifyCertificateCount < 2
	}, func(_, verifyConnectionCount uint64) bool {
		// not verification succeeded, so no connection was ever established
		return verifyConnectionCount == 0
	})
	dpsShutDown = test.New(t, dpsCfg, service.WithAuthHandler(ah))
	deferedDpsCleanUp = true
	err = ah.verify(ctx)
	require.NoError(t, err)

	deferedDpsCleanUp = false
	dpsShutDown()

	dpsCfg = test.MakeConfig(t)
	h := newTestRequestHandler(t, dpsCfg, defaultTestDpsHandlerConfig())
	h.StartDps(service.WithRequestHandler(h))
	defer h.StopDps()
	err = h.Verify(ctx)
	require.NoError(t, err)

	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_OFFLINE)
	require.NoError(t, err)
	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_ONLINE)
	require.NoError(t, err)
}
