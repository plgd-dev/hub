package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	authTest "github.com/go-ocf/cloud/authorization/provider"
	client "github.com/go-ocf/cloud/grpc-gateway/client"
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

	select {
	case res := <-h.res:
		t.Logf("res %+v\n", res)
		exp := test.ResourceLinkToPublishEvent(deviceID, 0, test.GetAllBackendResourceLinks())
		for _, link := range res.Links {
			link.InstanceId = 0
		}
		require.Equal(t, test.SortResources(exp.GetResourcePublished().GetLinks()), test.SortResources(res.Links))
	case <-time.After(TestTimeout):
		t.Error("timeout")
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
