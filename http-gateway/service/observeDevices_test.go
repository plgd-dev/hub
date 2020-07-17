package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/go-ocf/cloud/http-gateway/uri"
	testCfg "github.com/go-ocf/cloud/test/config"

	authTest "github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/cloud/http-gateway/service"
	"github.com/go-ocf/cloud/http-gateway/test"
	cloudTest "github.com/go-ocf/cloud/test"
	"github.com/go-ocf/kit/codec/json"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
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
		Status:    service.ToDevicesObservationEvent(client.DevicesObservationEvent_ONLINE),
	}
	testDeviceEvent(t, wsConn, expEvt)

	expEvt = service.DeviceEvent{
		DeviceIDs: nil,
		Status:    service.ToDevicesObservationEvent(client.DevicesObservationEvent_OFFLINE),
	}
	testDeviceEvent(t, wsConn, expEvt)

	//Second event
	shutdownDevSim()
	expEvt = service.DeviceEvent{
		DeviceIDs: []string{deviceID},
		Status:    service.ToDevicesObservationEvent(client.DevicesObservationEvent_OFFLINE),
	}
	testDeviceEvent(t, wsConn, expEvt)
}

func testDeviceEvent(t *testing.T, conn *websocket.Conn, expect service.DeviceEvent) {
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	evt := service.DeviceEvent{}
	err = json.Decode(message, &evt)
	require.NoError(t, err)
	require.Equal(t, expect, evt)
}

func GetDevicesObservationUri() string {
	return fmt.Sprintf("wss://localhost:%d%s", test.HTTP_GW_Port, uri.WsStartDevicesObservation)
}
