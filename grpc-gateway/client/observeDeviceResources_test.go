package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	authTest "github.com/go-ocf/cloud/authorization/provider"
	client "github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	test "github.com/go-ocf/cloud/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/stretchr/testify/require"
)

func TestObserveDeviceResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	c := NewTestClient(t)
	defer c.Close(context.Background())
	shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	h := makeTestDeviceResourcesObservationHandler()
	id, err := c.ObserveDeviceResources(ctx, deviceID, h)
	require.NoError(t, err)
	defer func() {
		err := c.StopObservingDevices(ctx, id)
		require.NoError(t, err)
	}()

LOOP:
	for {
		select {
		case res := <-h.res:
			t.Logf("res %+v\n", res)
			if res.Link.GetHref() == "/oic/d" {
				require.Equal(t, client.DeviceResourcesObservationEvent{
					Link: pb.ResourceLink{
						Href:       "/oic/d",
						Types:      []string{"oic.d.cloudDevice", "oic.wk.d"},
						Interfaces: []string{"oic.if.r", "oic.if.baseline"},
						DeviceId:   deviceID,
					},
					Event: client.DeviceResourcesObservationEvent_ADDED,
				}, res)
				break LOOP
			}
		case <-time.After(TestTimeout):
			t.Error("timeout")
			break LOOP
		}

	}
}

func makeTestDeviceResourcesObservationHandler() *testDeviceResourcesObservationHandler {
	return &testDeviceResourcesObservationHandler{res: make(chan client.DeviceResourcesObservationEvent, 100)}
}

type testDeviceResourcesObservationHandler struct {
	res chan client.DeviceResourcesObservationEvent
}

func (h *testDeviceResourcesObservationHandler) Handle(ctx context.Context, body client.DeviceResourcesObservationEvent) error {
	h.res <- body
	return nil
}

func (h *testDeviceResourcesObservationHandler) Error(err error) { fmt.Println(err) }

func (h *testDeviceResourcesObservationHandler) OnClose() {
	fmt.Println("devices observation was closed")
}
