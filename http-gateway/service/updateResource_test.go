package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/plgd-dev/kit/codec/json"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	cloudTest "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestUpdateResource(t *testing.T) {
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

	request := map[string]interface{}{
		"power": 111,
	}
	var response interface{}

	UpdateResource(t, deviceID, uri.Device+"/light/1", request, &response)
	require.Equal(t, nil, response)

	request["power"] = 0
	UpdateResource(t, deviceID, uri.Device+"/light/1", request, &response)
	require.Equal(t, nil, response)
}

func UpdateResource(t *testing.T, deviceID, uri string, request interface{}, response interface{}) {
	reqData, err := json.Encode(request)
	require.NoError(t, err)

	getReq := test.NewRequest("PUT", uri, bytes.NewReader(reqData)).
		DeviceId(deviceID).AuthToken(oauthTest.GetServiceToken(t)).Build()
	res := test.HTTPDo(t, getReq)
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	if len(b) > 0 {
		require.Equal(t, http.StatusOK, res.StatusCode)
		err = json.Decode(b, &response)
		require.NoError(t, err)
	} else {
		require.Equal(t, http.StatusNoContent, res.StatusCode)
	}
}

func TestUpdateResourceInvalidAttribute(t *testing.T) {
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

	request := map[string]interface{}{
		"power": "Test string",
	}
	reqData, err := json.Encode(request)
	require.NoError(t, err)

	getReq := test.NewRequest("PUT", uri.Device+"/light/1", bytes.NewReader(reqData)).
		DeviceId(deviceID).AuthToken(oauthTest.GetServiceToken(t)).Build()
	res := test.HTTPDo(t, getReq)
	defer res.Body.Close()
	var response interface{}
	err = json.ReadFrom(res.Body, &response)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
	exp := map[interface{}]interface{}{
		"err": "cannot update resource: cannot update resource /" + deviceID + "//light/1: rpc error: code = InvalidArgument desc = response from device",
	}
	require.Equal(t, exp, response)
}
