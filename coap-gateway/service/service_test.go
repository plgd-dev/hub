package service_test

import (
	"context"
	"crypto/tls"
	"testing"

	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	test "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type testDevsObs struct {
	err atomic.Error
	ch  chan client.DevicesObservationEvent
}

func (t *testDevsObs) Error(err error) {
	t.err.Store(err)
}

func (t *testDevsObs) Handle(ctx context.Context, event client.DevicesObservationEvent) error {
	t.ch <- event
	return nil
}

func (t *testDevsObs) OnClose() {}

func TestShutdownServiceWithDeviceIssue627(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	const services = service.SetUpServicesOAuth | service.SetUpServicesId | service.SetUpServicesResourceDirectory |
		service.SetUpServicesGrpcGateway | service.SetUpServicesResourceAggregate
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	coapShutdown := coapgwTest.SetUp(t)
	defer coapShutdown()

	grpcConn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = grpcConn.Close()
	}()
	grpcClient := client.New(pb.NewGrpcGatewayClient(grpcConn))

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, pb.NewGrpcGatewayClient(grpcConn), deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	ch := make(chan client.DevicesObservationEvent, 1000)

	v := testDevsObs{
		ch: ch,
	}

	observationID, err := grpcClient.ObserveDevices(ctx, &v)
	require.NoError(t, err)
	defer func(observationID string) {
		err := grpcClient.StopObservingDevices(ctx, observationID)
		require.NoError(t, err)
		require.NoError(t, v.err.Load())
	}(observationID)

	coapShutdown()

	for {
		select {
		case e := <-ch:
			if e.Event != client.DevicesObservationEvent_OFFLINE {
				continue
			}
			require.Len(t, e.DeviceIDs, 1)
			require.Equal(t, deviceID, e.DeviceIDs[0])
			return
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
		}
	}
}
