package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/kit/codec/json"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/service"
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

func TestUpdateDevice(t *testing.T) {
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

	request := service.UpdateDevice{
		ShadowSynchronization: service.ShadowSynchronization{
			Disabled: true,
		},
	}

	UpdateDevice(t, deviceID, uri.Device, "1", request)

	time.Sleep(time.Second)

	var d service.Device
	getDevice(t, deviceID, &d)
	require.True(t, d.Metadata.ShadowSynchronization.Disabled)
	request.ShadowSynchronization.Disabled = false
	UpdateDevice(t, deviceID, uri.Device, "2", request)
	time.Sleep(time.Second)
	getDevice(t, deviceID, &d)
	require.False(t, d.Metadata.ShadowSynchronization.Disabled)
}

func UpdateDevice(t *testing.T, deviceID, url, correlationID string, request interface{}) {
	reqData, err := json.Encode(request)
	require.NoError(t, err)

	getReq := test.NewRequest("PUT", url, bytes.NewReader(reqData)).
		DeviceId(deviceID).AuthToken(oauthTest.GetServiceToken(t)).Build()
	getReq.Header.Set(uri.CorrelationIDHeader, correlationID)
	res := test.HTTPDo(t, getReq)
	defer res.Body.Close()

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, res.StatusCode)

}
