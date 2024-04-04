package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	test "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestObserveDeviceResourcesPublish(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	c := NewTestClient(t)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	h := makeTestDeviceResourcesObservationHandler()
	id, err := c.ObserveDeviceResources(ctx, deviceID, h)
	require.NoError(t, err)
	defer func() {
		err := c.StopObservingDevices(id)
		require.NoError(t, err)
	}()

	select {
	case res := <-h.res:
		t.Logf("res %+v\n", res)
		pub, ok := res.(*events.ResourceLinksPublished)
		require.True(t, ok)
		exp := pbTest.ResourceLinkToPublishEvent(deviceID, "", test.GetAllBackendResourceLinks())
		test.CheckProtobufs(t, pbTest.CleanUpResourceLinksPublished(exp.GetResourcePublished(), false),
			pbTest.CleanUpResourceLinksPublished(pub, false), test.RequireToCheckFunc(require.Equal))
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

func (h *testDeviceResourcesObservationHandler) HandleResourcePublished(_ context.Context, val *events.ResourceLinksPublished) error {
	h.res <- val
	return nil
}

func (h *testDeviceResourcesObservationHandler) HandleResourceUnpublished(_ context.Context, val *events.ResourceLinksUnpublished) error {
	h.res <- val
	return nil
}

func (h *testDeviceResourcesObservationHandler) Error(err error) { fmt.Println(err) }

func (h *testDeviceResourcesObservationHandler) OnClose() {
	fmt.Println("devices observation was closed")
}
