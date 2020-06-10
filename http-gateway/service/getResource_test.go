package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"testing"

	authTest "github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/cloud/http-gateway/test"
	"github.com/go-ocf/cloud/http-gateway/uri"
	cloudTest "github.com/go-ocf/cloud/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	"github.com/go-ocf/kit/codec/json"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestGetResource(t *testing.T) {
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

	var response map[string]interface{}
	getResource(t, deviceID, uri.Device+"/light/1", &response)
	require.Equal(t, map[string]interface{}{"name": "Light", "power": uint64(0), "state": false}, response)
}

func TestGetResourceNotExist(t *testing.T) {
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

	getReq := test.NewRequest("GET", uri.Device+"/notExist", strings.NewReader("")).
		DeviceId(deviceID).AuthToken(authTest.UserToken).Build()
	res := test.HTTPDo(t, getReq)
	defer res.Body.Close()
	var response map[string]string
	err = json.ReadFrom(res.Body, &response)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, res.StatusCode)
	exp := map[string]string{
		"err": "cannot get resource: rpc error: code = NotFound desc = not found",
	}
	require.Equal(t, exp, response)
}

func getResource(t *testing.T, deviceID, uri string, data interface{}) {
	getReq := test.NewRequest("GET", uri, nil).
		DeviceId(deviceID).AuthToken(authTest.UserToken).Build()
	res := test.HTTPDo(t, getReq)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	err := json.ReadFrom(res.Body, &data)
	require.NoError(t, err)
}
