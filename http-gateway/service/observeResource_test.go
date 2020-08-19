package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	authTest "github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	cloudTest "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/plgd-dev/kit/codec/json"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const ConResource = "/oc/con"

func TestObserveResource(t *testing.T) {
	deviceID := cloudTest.MustFindDeviceByName(cloudTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), 2*test.TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)
	tearDown := cloudTest.SetUp(ctx, t)

	defer tearDown()

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: cloudTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer conn.Close()
	shutdownDevSim := cloudTest.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, cloudTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	webTearDown := test.SetUp(t)
	defer webTearDown()

	testResourceObservation(t, deviceID)
}

func testResourceObservation(t *testing.T, deviceID string) {

	time.Sleep(time.Second * 2)
	conn := webSocketConnection(t, GetResourceObservationUri(deviceID, ConResource))
	defer closeWebSocketConnection(t, conn)

	//first event
	t.Log("first event immediately after subscription")
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	evt := updateDeviceName{}
	err = json.Decode(message, &evt)
	require.NoError(t, err)
	require.Equal(t, "devsim-"+cloudTest.MustGetHostname(), evt.Name)

	//first update
	req := updateDeviceName{
		Name: "Test device name 1",
	}
	res := updateDeviceName{}
	UpdateResource(t, deviceID, uri.Device+ConResource, &req, &res)

	//second event
	t.Log("second event after first update")
	_, message, err = conn.ReadMessage()
	evt = updateDeviceName{}
	err = json.Decode(message, &evt)
	require.NoError(t, err)
	require.Equal(t, req.Name, evt.Name)

	//second update
	req = updateDeviceName{
		Name: "Test device name 2",
	}
	res = updateDeviceName{}
	UpdateResource(t, deviceID, uri.Device+ConResource, &req, &res)

	//third event
	t.Log("third event after second update")
	_, message, err = conn.ReadMessage()
	evt = updateDeviceName{}
	err = json.Decode(message, &evt)
	require.NoError(t, err)
	require.Equal(t, req.Name, evt.Name)

	//update back to old value
	req = updateDeviceName{
		Name: "devsim-" + cloudTest.MustGetHostname(),
	}
	res = updateDeviceName{}
	UpdateResource(t, deviceID, uri.Device+ConResource, &req, &res)
	time.Sleep(time.Second * 2)
}

func GetResourceObservationUri(deviceID, href string) string {
	return fmt.Sprintf("wss://localhost:%d%s/%s%s", test.HTTP_GW_Port, uri.WSDevices, deviceID, href)
}

func webSocketConnection(t *testing.T, uri string) *websocket.Conn {
	header := make(http.Header)
	header.Add("Authorization", fmt.Sprintf("Bearer %s", authTest.UserToken))
	d := websocket.DefaultDialer
	d.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conn, _, err := d.Dial(uri, header)
	require.NoError(t, err)
	return conn
}

func closeWebSocketConnection(t *testing.T, ws *websocket.Conn) {
	err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	require.NoError(t, err)
}

//not map all property from response
type updateDeviceName struct {
	Name string `json:"n"`
}
