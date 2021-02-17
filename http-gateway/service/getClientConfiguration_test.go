package service_test

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	oauthTest "github.com/plgd-dev/cloud/oauth-server/test"
	cloudTest "github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/plgd-dev/kit/codec/json"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/stretchr/testify/require"
)

func TestGetClientConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), test.TestTimeout)
	defer cancel()

	os.Setenv("SERVICE_CLIENT_CONFIGURATION_CLOUDURL", "coaps+tcp://"+config.GW_HOST)
	defer os.Unsetenv("SERVICE_CLIENT_CONFIGURATION_CLOUDURL")
	tearDown := cloudTest.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	webTearDown := test.SetUp(t)
	defer webTearDown()

	var response map[string]interface{}
	getReq := test.NewRequest("GET", uri.ClientConfiguration, nil).Build()
	res := test.HTTPDo(t, getReq)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	err := json.ReadFrom(res.Body, &response)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"cloud_url": "coaps+tcp://localhost:20002"}, response)
}
