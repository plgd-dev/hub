package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	test "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
)

func TestObserveDeviceResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	c := NewTestClient(t)
	defer c.Close(context.Background())
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
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
		pub, ok := res.(*events.ResourceLinksPublished)
		require.True(t, ok)
		exp := test.ResourceLinkToPublishEvent(deviceID, "", test.GetAllBackendResourceLinks())
		test.CheckProtobufs(t, test.CleanUpResourceLinksPublished(exp.GetResourcePublished()), test.CleanUpResourceLinksPublished(pub), test.RequireToCheckFunc(require.Equal))
	case <-time.After(TestTimeout):
		t.Error("timeout")
	}
}

func makeTestDeviceResourcesObservationHandler() *testDeviceResourcesObservationHandler {
	return &testDeviceResourcesObservationHandler{res: make(chan interface{}, 100)}
}

type testDeviceResourcesObservationHandler struct {
	res chan interface{}
}

func (h *testDeviceResourcesObservationHandler) HandleResourcePublished(ctx context.Context, val *events.ResourceLinksPublished) error {
	h.res <- val
	return nil
}

func (h *testDeviceResourcesObservationHandler) HandleResourceUnpublished(ctx context.Context, val *events.ResourceLinksUnpublished) error {
	h.res <- val
	return nil
}

func (h *testDeviceResourcesObservationHandler) Error(err error) { fmt.Println(err) }

func (h *testDeviceResourcesObservationHandler) OnClose() {
	fmt.Println("devices observation was closed")
}
