package service_test

import (
	"net/http"
	"testing"

	"github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
	"github.com/plgd-dev/kit/codec/json"
	"github.com/stretchr/testify/require"
)

func TestRequestHandler_getUserInfo(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	getReq := test.NewRequest(http.MethodGet, uri.UserInfo, nil).Build()
	res := test.HTTPDo(t, getReq, false)
	defer res.Body.Close()

	var body map[string]string
	err := json.ReadFrom(res.Body, &body)
	require.NoError(t, err)
	require.Equal(t, "1", body["sub"])
}
