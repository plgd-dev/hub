package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"testing"

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

func TestDeviceReboot(t *testing.T) {
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

	req := test.NewRequest("POST", uri.DeviceReboot, strings.NewReader("")).DeviceId(deviceID).AuthToken(oauthTest.GetServiceToken(t)).Build()
	res := test.HTTPDo(t, req)
	require.Equal(t, http.StatusNotFound, res.StatusCode)
}
