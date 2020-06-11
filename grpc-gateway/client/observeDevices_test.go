package client_test

import (
	"context"
	"fmt"
	"testing"

	authTest "github.com/go-ocf/cloud/authorization/provider"
	client "github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/stretchr/testify/require"
)

func TestObserveDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	c := NewTestClient(t)
	defer c.Close(context.Background())
	shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())

	h := makeTestDevicesObservationHandler()
	id, err := c.ObserveDevices(ctx, h)
	require.NoError(t, err)
	defer func() {
		c.StopObservingDevices(ctx, id)
	}()

	res := <-h.res
	require.Equal(t, client.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    client.DevicesObservationEvent_REGISTERED,
	}, res)
	res = <-h.res
	require.Equal(t, client.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    client.DevicesObservationEvent_ONLINE,
	}, res)

	shutdownDevSim()
	res = <-h.res
	require.Equal(t, client.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    client.DevicesObservationEvent_OFFLINE,
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
