package client_test

import (
	"context"
	"fmt"
	"testing"

	client "github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/stretchr/testify/require"
)

func TestObserveDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	c := NewTestClient(t)
	defer c.Close(context.Background())
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())

	h := makeTestDevicesObservationHandler()
	id, err := c.ObserveDevices(ctx, h)
	require.NoError(t, err)
	defer func() {
		c.StopObservingDevices(ctx, id)
	}()

	res := <-h.res
	require.Equal(t, client.DevicesObservationEvent{
		DeviceIDs: []string{deviceID},
		Event:     client.DevicesObservationEvent_REGISTERED,
	}, res)

	res = <-h.res
	require.Equal(t, client.DevicesObservationEvent{
		DeviceIDs: nil,
		Event:     client.DevicesObservationEvent_UNREGISTERED,
	}, res)

	res = <-h.res
	require.Equal(t, client.DevicesObservationEvent{
		DeviceIDs: []string{deviceID},
		Event:     client.DevicesObservationEvent_ONLINE,
	}, res)

	res = <-h.res
	require.Equal(t, client.DevicesObservationEvent{
		DeviceIDs: nil,
		Event:     client.DevicesObservationEvent_OFFLINE,
	}, res)

	shutdownDevSim()
	res = <-h.res
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

func (h *testDevicesObservationHandler) Handle(ctx context.Context, body client.DevicesObservationEvent) error {
	h.res <- body
	return nil
}

func (h *testDevicesObservationHandler) Error(err error) { fmt.Println(err) }

func (h *testDevicesObservationHandler) OnClose() { fmt.Println("devices observation was closed") }
