package service_test

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/kit/codec/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"context"

	authTest "github.com/go-ocf/cloud/authorization/provider"
	grpcTest "github.com/go-ocf/cloud/grpc-gateway/test"
	"github.com/go-ocf/cloud/http-gateway/test"
	"github.com/go-ocf/cloud/http-gateway/uri"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/stretchr/testify/require"
)

func TestGetDevices(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), test.TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)
	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(grpcTest.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: grpcTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer conn.Close()
	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c, deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	webTearDown := test.NewTestHTTPGW(t, test.NewTestBackendConfig().String())
	defer webTearDown()

	var response []interface{}
	getDevices(t, &response)
	require.Len(t, response, 1)
	require.Equal(t, test.GetDeviceRepresentation(deviceID, grpcTest.TestDeviceName), test.CleanUpDeviceRepresentation(response[0]))
}

func getDevices(t *testing.T, response interface{}) {
	req := test.NewRequest(http.MethodGet, uri.Devices, nil).AuthToken(authTest.UserToken).Build()
	req.Header.Set("Request-Timeout", "1")

	res := test.HTTPDo(t, req)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	err := json.ReadFrom(res.Body, response)
	require.NoError(t, err)
}
