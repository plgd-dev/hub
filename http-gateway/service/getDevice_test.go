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

	authTest "github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	cloudTest "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/stretchr/testify/require"
)

func TestGetDevice(t *testing.T) {
	deviceID := cloudTest.MustFindDeviceByName(cloudTest.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), test.TestTimeout)
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
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)
	tearDown := cloudTest.SetUp(ctx, t)
	defer tearDown()

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
		DeviceId("notExist").AuthToken(authTest.UserToken).Build()
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
	req := test.NewRequest(http.MethodGet, uri.Device, nil).DeviceId(deviceID).AuthToken(authTest.UserToken).Build()
	req.Header.Set("Request-Timeout", "1")

	res := test.HTTPDo(t, req)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	err := json.ReadFrom(res.Body, response)
	require.NoError(t, err)
}
