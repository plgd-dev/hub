package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/go-ocf/kit/codec/json"

	authTest "github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/cloud/http-gateway/test"
	"github.com/go-ocf/cloud/http-gateway/uri"
	cloudTest "github.com/go-ocf/cloud/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestUpdateResource(t *testing.T) {
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
	shutdownDevSim := cloudTest.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, cloudTest.GetAllBackendResourceLinks())
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
		DeviceId(deviceID).AuthToken(authTest.UserToken).Build()
	res := test.HTTPDo(t, getReq)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	b, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	if len(b) > 0 {
		err = json.Decode(b, &response)
		require.NoError(t, err)
	}
}

func TestUpdateResourceInvalidAttribute(t *testing.T) {
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
	shutdownDevSim := cloudTest.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, cloudTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	webTearDown := test.SetUp(t)
	defer webTearDown()

	request := map[string]interface{}{
		"power": "Test string",
	}
	reqData, err := json.Encode(request)
	require.NoError(t, err)

	getReq := test.NewRequest("PUT", uri.Device+"/light/1", bytes.NewReader(reqData)).
		DeviceId(deviceID).AuthToken(authTest.UserToken).Build()
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
