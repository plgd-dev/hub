package service_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/plgd-dev/cloud/oauth-server/service"
	"github.com/plgd-dev/cloud/oauth-server/test"
	"github.com/plgd-dev/cloud/oauth-server/uri"
	"github.com/plgd-dev/kit/codec/json"
	"github.com/stretchr/testify/require"
)

func TestRequestHandler_authorize(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()
	Authorize(t, service.ClientUI, "https://localhost:3000", http.StatusTemporaryRedirect)
	Authorize(t, service.ClientUI, "", http.StatusOK)
	Authorize(t, "badClient", "", http.StatusBadRequest)
	// service doesn't allows use authorization_code grant type
	Authorize(t, service.ClientService, "", http.StatusBadRequest)
}

func Authorize(t *testing.T, clientID string, redirectURI string, statusCode int) string {
	u, err := url.Parse(uri.Authorize)
	require.NoError(t, err)
	q, err := url.ParseQuery(u.RawQuery)
	require.NoError(t, err)
	q.Add(uri.ClientIDQueryKey, clientID)
	if redirectURI != "" {
		q.Add(uri.RedirectURIQueryKey, redirectURI)
		q.Add(uri.StateQueryKey, "1")
	}
	u.RawQuery = q.Encode()
	getReq := test.NewRequest(http.MethodGet, u.String(), nil).Build()
	res := test.HTTPDo(t, getReq, false)
	defer res.Body.Close()
	require.Equal(t, statusCode, res.StatusCode)
	if res.StatusCode == http.StatusTemporaryRedirect {
		loc, err := res.Location()
		require.NoError(t, err)
		code := loc.Query().Get(uri.CodeQueryKey)
		require.NotEmpty(t, code)
		return code
	}
	if res.StatusCode == http.StatusOK {
		var body map[string]string
		err := json.ReadFrom(res.Body, &body)
		require.NoError(t, err)
		code := body[uri.CodeQueryKey]
		require.NotEmpty(t, code)
		return code
	}

	return ""
}
