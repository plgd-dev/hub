package service_test

import (
	"crypto/tls"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/grpc"

	"github.com/plgd-dev/kit/codec/json"
	"google.golang.org/grpc/credentials"

	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	cloudTest "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
)

func TestGetDevice(t *testing.T) {
	deviceID := cloudTest.MustFindDeviceByName(cloudTest.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), test.TestTimeout)
	defer cancel()

	tearDown := cloudTest.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: cloudTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer conn.Close()
	deviceID, shutdownDevSim := cloudTest.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, cloudTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	webTearDown := test.SetUp(t)
	defer webTearDown()

	var d interface{}
	getDevice(t, deviceID, &d)
	require.Equal(t, test.GetDeviceRepresentation(deviceID, cloudTest.TestDeviceName), test.CleanUpDeviceRepresentation(d))
}

func TestGetDeviceNotExist(t *testing.T) {
	deviceID := cloudTest.MustFindDeviceByName(cloudTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), test.TestTimeout)
	defer cancel()

	tearDown := cloudTest.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: cloudTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer conn.Close()
	deviceID, shutdownDevSim := cloudTest.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, cloudTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	webTearDown := test.SetUp(t)
	defer webTearDown()

	getReq := test.NewRequest("GET", uri.Device, strings.NewReader("")).
		DeviceId("notExist").AuthToken(oauthTest.GetServiceToken(t)).Build()
	res := test.HTTPDo(t, getReq)
	defer res.Body.Close()
	var response interface{}
	err = json.ReadFrom(res.Body, &response)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, res.StatusCode)
	exp := map[interface{}]interface{}{
		"err": "cannot get device: rpc error: code = NotFound desc = not found",
	}
	require.Equal(t, exp, response)
}

func getDevice(t *testing.T, deviceID string, response interface{}) {
	req := test.NewRequest(http.MethodGet, uri.Device, nil).DeviceId(deviceID).AuthToken(oauthTest.GetServiceToken(t)).Build()
	req.Header.Set("Request-Timeout", "1")

	res := test.HTTPDo(t, req)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	err := json.ReadFrom(res.Body, response)
	require.NoError(t, err)
}
