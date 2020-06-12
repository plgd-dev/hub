package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
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

func TestDeviceResourcesObservation(t *testing.T) {
	deviceID := cloudTest.MustFindDeviceByName(cloudTest.TestDeviceName)
	//set up
	ctx, cancel := context.WithTimeout(context.Background(), 2*test.TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)
	tearDown := cloudTest.SetUp(ctx, t)
	defer tearDown()
	webTearDown := test.SetUp(t)
	defer webTearDown()

	//onboard
	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: cloudTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer conn.Close()
	shutdownDevSim := cloudTest.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, cloudTest.GetAllBackendResourceLinks())

	// create web socket connection
	wsConn := webSocketConnection(t, GetDeviceResourcesObservationUri(deviceID))
	defer closeWebSocketConnection(t, wsConn)

	//read messages
	received := sync.Map{}
	go readMessage(t, wsConn, &received)

	//ofboard
	shutdownDevSim()
}

func readMessage(t *testing.T, conn *websocket.Conn, messages *sync.Map) {
	for {
		tp, message, err := conn.ReadMessage()
		if tp == websocket.CloseMessage {
			return
		}
		if err != nil {
			return
		}
		var event service.DeviceResourceObservationEvent
		err = json.Decode(message, &event)
		require.NoError(t, err)
		if event.Event == service.ToDeviceResourcesObservationEvent(client.DeviceResourcesObservationEvent_ADDED) {
			messages.Store(event.Resource.Href, event)
		} else if event.Event == service.ToDeviceResourcesObservationEvent(client.DeviceResourcesObservationEvent_REMOVED) {
			messages.Delete(event.Resource.Href)
		}
	}
}

func GetDeviceResourcesObservationUri(deviceID string) string {
	return fmt.Sprintf("wss://localhost:%d%s/%s", test.HTTP_GW_Port, uri.WSDevices, deviceID)
}
