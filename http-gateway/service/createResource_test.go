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
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	cloudTest "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestCreateResource(t *testing.T) {
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

	CreateResource(t, deviceID, uri.Device+"/oic/d", request, &response, http.StatusForbidden)
	require.Equal(t, nil, response)
}

func CreateResource(t *testing.T, deviceID, uri string, request interface{}, response interface{}, statusCode int) {
	reqData, err := json.Encode(request)
	require.NoError(t, err)

	getReq := test.NewRequest(http.MethodPost, uri, bytes.NewReader(reqData)).
		DeviceId(deviceID).AuthToken(oauthTest.GetServiceToken(t)).Build()
	res := test.HTTPDo(t, getReq)
	defer res.Body.Close()

	require.Equal(t, statusCode, res.StatusCode)
	if http.StatusOK == res.StatusCode {
		b, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		err = json.Decode(b, &response)
		require.NoError(t, err)
	}
}
