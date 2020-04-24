package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/go-ocf/cloud/http-gateway/uri"

	authTest "github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	grpcTest "github.com/go-ocf/cloud/grpc-gateway/test"
	"github.com/go-ocf/cloud/http-gateway/service"
	"github.com/go-ocf/cloud/http-gateway/test"
	"github.com/go-ocf/kit/codec/json"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestObserveDevices(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), 2*test.TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)
	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()
	webTearDown := test.NewTestHTTPGW(t, test.NewTestBackendConfig().String())
	defer webTearDown()

	testObserveDevices(ctx, t, deviceID)
}

func testObserveDevices(ctx context.Context, t *testing.T, deviceID string) {

	wsConn := webSocketConnection(t, GetDevicesObservationUri())
	defer closeWebSocketConnection(t, wsConn)

	//first event
	conn, err := grpc.Dial(grpcTest.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: grpcTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer conn.Close()
	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c, deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())

	expEvt := service.DeviceEvent{
		DeviceId: deviceID,
		Status:   service.ToDevicesObservationEvent(client.DevicesObservationEvent_ONLINE),
	}
	testDeviceEvent(t, wsConn, expEvt)

	//Second event
	shutdownDevSim()
	expEvt = service.DeviceEvent{
		DeviceId: deviceID,
		Status:   service.ToDevicesObservationEvent(client.DevicesObservationEvent_OFFLINE),
	}
	testDeviceEvent(t, wsConn, expEvt)
}

func testDeviceEvent(t *testing.T, conn *websocket.Conn, expect service.DeviceEvent) {
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	evt := service.DeviceEvent{}
	err = json.Decode(message, &evt)
	require.NoError(t, err)
	require.Equal(t, expect.DeviceId, evt.DeviceId)
	require.Equal(t, expect.Status, evt.Status)
}

func GetDevicesObservationUri() string {
	return fmt.Sprintf("wss://localhost:%d%s", test.HTTP_GW_Port, uri.WsStartDevicesObservation)
}
