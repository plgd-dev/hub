package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/message/status"
	"github.com/plgd-dev/go-coap/v3/mux"
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
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type testDpsHandler interface {
	service.RequestHandler
	Cfg() service.Config
	StartDps(opts ...service.Option)
	RestartDps(opts ...service.Option)
	StopDps()
	Verify(ctx context.Context) error
	Logf(template string, args ...interface{})
}

func testProvisioningWithDPSHandler(t *testing.T, h testDpsHandler, timeout time.Duration, onboardingOpts ...test.Option) {
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|
		hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId|hubTestService.SetUpServicesCoapGateway|hubTestService.SetUpServicesResourceAggregate|
		hubTestService.SetUpServicesGrpcGateway)
	defer hubShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
	deviceID, shutdownSim := test.OnboardDpsSim(ctx, t, c, deviceID, h.Cfg().APIs.COAP.Addr, test.TestDevsimResources, onboardingOpts...)
	defer shutdownSim()

	err = h.Verify(ctx)
	require.NoError(t, err)

	err = test.WaitForDeviceStatus(t, subClient, deviceID, subID, corID, commands.Connection_ONLINE)
	require.NoError(t, err)
}

type testRequestHandlerWithDisconnect struct {
	test.RequestHandlerWithDps
	disconnectTime                 atomic.Bool
	disconnectOwnership            atomic.Bool
	disconnectedCredentials        atomic.Bool
	disconnectedACLs               atomic.Bool
	disconnectedCloudConfiguration atomic.Bool
	r                              service.RequestHandle
}

func newTestRequestHandlerWithDisconnect(t *testing.T, dpsCfg service.Config) *testRequestHandlerWithDisconnect {
	return &testRequestHandlerWithDisconnect{
		RequestHandlerWithDps: test.MakeRequestHandlerWithDps(t, dpsCfg),
	}
}

func (h *testRequestHandlerWithDisconnect) DefaultHandler(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	return h.r.DefaultHandler(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithDisconnect) ProcessPlgdTime(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if h.disconnectTime.CompareAndSwap(false, true) {
		err := session.Close()
		require.NoError(h.T(), err)
		return nil, status.Errorf(service.NewMessageWithCode(coapCodes.ServiceUnavailable), "ProcessPlgdTime: disconnect")
	}
	return h.r.ProcessPlgdTime(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithDisconnect) ProcessOwnership(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if h.disconnectOwnership.CompareAndSwap(false, true) {
		err := session.Close()
		require.NoError(h.T(), err)
		return nil, status.Errorf(service.NewMessageWithCode(coapCodes.ServiceUnavailable), "ProcessOwnership: disconnect")
	}
	return h.r.ProcessOwnership(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithDisconnect) ProcessCredentials(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if h.disconnectedCredentials.CompareAndSwap(false, true) {
		err := session.Close()
		require.NoError(h.T(), err)
		return nil, status.Errorf(service.NewMessageWithCode(coapCodes.ServiceUnavailable), "ProcessCredentials: disconnect")
	}
	return h.r.ProcessCredentials(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithDisconnect) ProcessACLs(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if h.disconnectedACLs.CompareAndSwap(false, true) {
		err := session.Close()
		require.NoError(h.T(), err)
		return nil, status.Errorf(service.NewMessageWithCode(coapCodes.ServiceUnavailable), "ProcessACLs: disconnect")
	}
	return h.r.ProcessACLs(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithDisconnect) ProcessCloudConfiguration(ctx context.Context, req *mux.Message, session *service.Session, linkedHubs []*service.LinkedHub, group *service.EnrollmentGroup) (*pool.Message, error) {
	if h.disconnectedCloudConfiguration.CompareAndSwap(false, true) {
		err := session.Close()
		require.NoError(h.T(), err)
		return nil, status.Errorf(service.NewMessageWithCode(coapCodes.ServiceUnavailable), "ProcessCloudConfiguration: disconnect")
	}
	return h.r.ProcessCloudConfiguration(ctx, req, session, linkedHubs, group)
}

func (h *testRequestHandlerWithDisconnect) Verify(ctx context.Context) error {
	logCounter := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("unexpected counters disconnectTime=%v disconnectedOwnership=%v disconnectedCredentials=%v disconnectedACLs=%v disconnectedCloudConfiguration=%v",
				h.disconnectTime.Load(),
				h.disconnectOwnership.Load(),
				h.disconnectedCredentials.Load(),
				h.disconnectedACLs.Load(),
				h.disconnectedCloudConfiguration.Load())
		case <-time.After(time.Second):
			logCounter++
			if logCounter%3 == 0 {
				h.Logf("disconnectTime=%v disconnectedOwnership=%v disconnectedCredentials=%v disconnectedACLs=%v disconnectedCloudConfiguration=%v",
					h.disconnectTime.Load(),
					h.disconnectOwnership.Load(),
					h.disconnectedCredentials.Load(),
					h.disconnectedACLs.Load(),
					h.disconnectedCloudConfiguration.Load())
			}
		}
		if h.disconnectTime.Load() && h.disconnectOwnership.Load() &&
			h.disconnectedCredentials.Load() &&
			h.disconnectedACLs.Load() &&
			h.disconnectedCloudConfiguration.Load() {
			return nil
		}
	}
}

func TestProvisioningWithDisconnect(t *testing.T) {
	dpsCfg := test.MakeConfig(t)
	rh := newTestRequestHandlerWithDisconnect(t, dpsCfg)
	testProvisioningWithDPSHandler(t, rh, time.Minute)
}
