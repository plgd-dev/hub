package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"testing"

	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/test/config"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"

	"github.com/gorilla/websocket"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/service"
	"github.com/plgd-dev/cloud/http-gateway/test"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	cloudTest "github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/kit/codec/json"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestDeviceResourcesObservation(t *testing.T) {
	deviceID := cloudTest.MustFindDeviceByName(cloudTest.TestDeviceName)
	//set up
	ctx, cancel := context.WithTimeout(context.Background(), 2*test.TestTimeout)
	defer cancel()

	tearDown := cloudTest.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))
	webTearDown := test.SetUp(t)
	defer webTearDown()

	//onboard
	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: cloudTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer conn.Close()
	deviceID, shutdownDevSim := cloudTest.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, cloudTest.GetAllBackendResourceLinks())

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
		if event.Event == "added" {
			messages.Store("added", event)
		} else if event.Event == "removed" {
			messages.Delete("deleted")
		}
	}
}

func GetDeviceResourcesObservationUri(deviceID string) string {
	return fmt.Sprintf("wss://%v%v/%s", config.HTTP_GW_HOST, uri.WSDevices, deviceID)
}
