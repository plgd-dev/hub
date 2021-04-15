package service_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	rdTest "github.com/plgd-dev/cloud/resource-directory/test"
	cloudTest "github.com/plgd-dev/cloud/test"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/kit/codec/json"
	"github.com/stretchr/testify/require"
)

func TestGetClientConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), test.TestTimeout)
	defer cancel()

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

	data, err := json.Encode(rdTest.MakeConfig(t).ExposedCloudConfiguration.ToProto())
	require.NoError(t, err)
	var exp map[string]interface{}
	err = json.Decode(data, &exp)
	require.NoError(t, err)
	require.NotEmpty(t, response["cloud_certificate_authorities"])
	delete(response, "cloud_certificate_authorities")
	require.Equal(t, exp, response)
}
