package service_test

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/plgd-dev/kit/codec/json"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	oauthTest "github.com/plgd-dev/cloud/oauth-server/test"
	cloudTest "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestDeleteResource(t *testing.T) {
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

	var response interface{}

	err = DeleteResource(t, deviceID, uri.Device+"/oic/d", &response, http.StatusForbidden)
	require.NoError(t, err)

	/*
		err = DeleteResource(t, deviceID, uri.Device+"/light/1", &response, http.StatusMethodNotAllowed)
		require.NoError(t, err)
	*/
}

func DeleteResource(t *testing.T, deviceID, uri string, response interface{}, statusCode int) error {
	getReq := test.NewRequest(http.MethodDelete, uri, nil).
		DeviceId(deviceID).AuthToken(oauthTest.GetServiceToken(t)).Build()
	res := test.HTTPDo(t, getReq)
	defer res.Body.Close()
	require.Equal(t, statusCode, res.StatusCode)
	b, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	if len(b) > 0 {
		err = json.Decode(b, &response)
		require.NoError(t, err)
	}
	return nil
}
