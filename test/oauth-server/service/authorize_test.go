package service_test

import (	
	"net/http"
	"net/url"
	"testing"

	"github.com/plgd-dev/hub/test/config"
	"github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/hub/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

func TestRequestHandler_authorize(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()
	getAuthorize(t, test.ClientTest, "nonse", "https://localhost:3000", "", "", http.StatusTemporaryRedirect)
	getAuthorize(t, test.ClientTest, "", "", "", "", http.StatusOK)
}

func getAuthorize(t *testing.T, clientID, nonce, redirectURI, deviceID, scope string, statusCode int) string {
	u, err := url.Parse(uri.Authorize)
	require.NoError(t, err)
	q, err := url.ParseQuery(u.RawQuery)
	require.NoError(t, err)
	q.Add(uri.ClientIDKey, clientID)
	if redirectURI != "" {
		q.Add(uri.RedirectURIKey, redirectURI)
		q.Add(uri.StateKey, "1")
	}
	if nonce != "" {
		q.Add(uri.NonceKey, nonce)
	}
	if deviceID != "" {
		q.Add(uri.DeviceId, deviceID)
	}
	if scope != "" {
		q.Add(uri.ScopeKey, scope)
	}

	u.RawQuery = q.Encode()
	getReq := test.NewRequest(http.MethodGet, config.OAUTH_SERVER_HOST, u.String(), nil).Build()
	res := test.HTTPDo(t, getReq, false)
	defer func() {
		_ = res.Body.Close()
	}()
	require.Equal(t, statusCode, res.StatusCode)
	if res.StatusCode == http.StatusTemporaryRedirect {
		loc, err := res.Location()
		require.NoError(t, err)
		code := loc.Query().Get(uri.CodeKey)
		require.NotEmpty(t, code)
		return code
	}
	if res.StatusCode == http.StatusOK {
		var body map[string]string
		err := json.ReadFrom(res.Body, &body)
		require.NoError(t, err)
		code := body[uri.CodeKey]
		require.NotEmpty(t, code)
		return code
	}

	return ""
}
