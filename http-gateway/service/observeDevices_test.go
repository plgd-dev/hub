package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/plgd-dev/cloud/http-gateway/uri"
	testCfg "github.com/plgd-dev/cloud/test/config"

	authTest "github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/service"
	"github.com/plgd-dev/cloud/http-gateway/test"
	cloudTest "github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/kit/codec/json"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestObserveDevices(t *testing.T) {
	deviceID := cloudTest.MustFindDeviceByName(cloudTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), 2*test.TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)
	tearDown := cloudTest.SetUp(ctx, t)
	defer tearDown()
	webTearDown := test.SetUp(t)
	defer webTearDown()

	testObserveDevices(ctx, t, deviceID)
}

func testObserveDevices(ctx context.Context, t *testing.T, deviceID string) {
	//first event
	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: cloudTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer conn.Close()
	shutdownDevSim := cloudTest.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, cloudTest.GetAllBackendResourceLinks())

	wsConn := webSocketConnection(t, GetDevicesObservationUri())
	defer closeWebSocketConnection(t, wsConn)
	expEvt := service.DeviceEvent{
		DeviceIDs: []string{deviceID},
		Status:    service.ToDevicesObservationEvent(client.DevicesObservationEvent_REGISTERED),
	}
	testDeviceEvent(t, wsConn, expEvt)

	expEvt = service.DeviceEvent{
		DeviceIDs: []string{deviceID},
		Status:    service.ToDevicesObservationEvent(client.DevicesObservationEvent_ONLINE),
	}
	testDeviceEvent(t, wsConn, expEvt)

	//Second event
	shutdownDevSim()
	evt := testDeviceRecvEvent(t, wsConn)
	require.True(t, evt.Status == service.ToDevicesObservationEvent(client.DevicesObservationEvent_OFFLINE) || evt.Status == service.ToDevicesObservationEvent(client.DevicesObservationEvent_UNREGISTERED))
	expEvt = service.DeviceEvent{
		DeviceIDs: []string{deviceID},
		Status:    evt.Status,
	}
	require.Equal(t, expEvt, evt)
}

func testDeviceRecvEvent(t *testing.T, conn *websocket.Conn) service.DeviceEvent {
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	evt := service.DeviceEvent{}
	err = json.Decode(message, &evt)
	require.NoError(t, err)
	return evt
}

func testDeviceEvent(t *testing.T, conn *websocket.Conn, expect service.DeviceEvent) {
	evt := testDeviceRecvEvent(t, conn)
	require.Equal(t, expect, evt)
}

func GetDevicesObservationUri() string {
	return fmt.Sprintf("wss://localhost:%d%s", test.HTTP_GW_Port, uri.WsStartDevicesObservation)
}
