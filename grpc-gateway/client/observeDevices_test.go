package client_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	client "github.com/plgd-dev/hub/v2/grpc-gateway/client"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestObserveDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	c := grpcgwTest.NewTestClient(t)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())

	h := makeTestDevicesObservationHandler()
	id, err := c.ObserveDevices(ctx, h)
	require.NoError(t, err)
	defer func() {
		err := c.StopObservingDevices(id)
		require.NoError(t, err)
	}()

	var res client.DevicesObservationEvent

	select {
	case res = <-h.res:
	case <-ctx.Done():
		require.NoError(t, errors.New("timeout"))
	}
	require.Equal(t, client.DevicesObservationEvent{
		DeviceIDs: []string{deviceID},
		Event:     client.DevicesObservationEvent_REGISTERED,
	}, res)

	select {
	case res = <-h.res:
	case <-ctx.Done():
		require.NoError(t, errors.New("timeout"))
	}
	require.Equal(t, client.DevicesObservationEvent{
		DeviceIDs: []string{deviceID},
		Event:     client.DevicesObservationEvent_ONLINE,
	}, res)

	shutdownDevSim()
	select {
	case res = <-h.res:
	case <-ctx.Done():
		require.NoError(t, errors.New("timeout"))
	}
	require.True(t, res.Event == client.DevicesObservationEvent_OFFLINE || res.Event == client.DevicesObservationEvent_UNREGISTERED)
	require.Equal(t, client.DevicesObservationEvent{
		DeviceIDs: []string{deviceID},
		Event:     res.Event,
	}, res)
}

func makeTestDevicesObservationHandler() *testDevicesObservationHandler {
	return &testDevicesObservationHandler{res: make(chan client.DevicesObservationEvent, 10)}
}

type testDevicesObservationHandler struct {
	res chan client.DevicesObservationEvent
}

func (h *testDevicesObservationHandler) Handle(_ context.Context, body client.DevicesObservationEvent) error {
	h.res <- body
	return nil
}

func (h *testDevicesObservationHandler) Error(err error) { fmt.Println(err) }

func (h *testDevicesObservationHandler) OnClose() { fmt.Println("devices observation was closed") }
