package client_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/plgd-dev/device/v2/schema/configuration"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DecodeFunc = func(interface{}) error

func TestObservingResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	c := grpcgwTest.NewTestClient(t)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	h := makeTestObservationHandler()
	id, err := c.ObserveResource(ctx, deviceID, configuration.ResourceURI, h)
	require.NoError(t, err)
	defer func() {
		errS := c.StopObservingResource(id)
		require.NoError(t, errS)
	}()

	name := "observe simulator"
	err = c.UpdateResource(ctx, deviceID, configuration.ResourceURI, map[string]interface{}{"n": name}, nil)
	require.NoError(t, err)

	var d OcCon
	res := <-h.res
	err = res(&d)
	require.NoError(t, err)
	assert.Equal(t, test.TestDeviceName, d.Name)
	res = <-h.res
	err = res(&d)
	require.NoError(t, err)
	require.Equal(t, name, d.Name)

	err = c.UpdateResource(ctx, deviceID, configuration.ResourceURI, map[string]interface{}{"n": test.TestDeviceName}, nil)
	require.NoError(t, err)
}

func makeTestObservationHandler() *testObservationHandler {
	return &testObservationHandler{res: make(chan DecodeFunc, 10)}
}

type OcCon struct {
	Name string `json:"n"`
}

type testObservationHandler struct {
	res chan DecodeFunc
}

func (h *testObservationHandler) Handle(_ context.Context, body DecodeFunc) {
	h.res <- body
}

func (h *testObservationHandler) Error(err error) {
	fmt.Println(err)
}

func (h *testObservationHandler) OnClose() {
	fmt.Println("Observation was closed")
}
