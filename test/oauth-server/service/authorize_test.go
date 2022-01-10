package service_test

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

func TestRequestHandler_authorize(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()
	getAuthorize(t, test.ClientTest, "nonse", "https://localhost:3000", "", "", "", http.StatusFound, false, false)
	getAuthorize(t, test.ClientTest, "", "", "", "", "", http.StatusOK, false, false)
	getAuthorize(t, test.ClientTestRequiredParams, "nonce", "http://localhost:1234", "", "", "", http.StatusBadRequest, false, false)
	getAuthorize(t, test.ClientTestRequiredParams, "nonce", "http://localhost:7777", "", "", "code", http.StatusFound, true, false)
	getAuthorize(t, test.ClientTestRequiredParams, "nonce", "http://localhost:7777", "", "wrong_scope", "code", http.StatusFound, true, false)
	getAuthorize(t, test.ClientTestRequiredParams, "nonce", "http://localhost:7777", "", "r:* wrong_scope", "code", http.StatusFound, true, false)
	getAuthorize(t, test.ClientTestRequiredParams, "nonce", "http://localhost:7777", "", "offline_access", "code", http.StatusFound, false, false)
	getAuthorize(t, test.ClientTestRequiredParams, "nonce", "http://localhost:7777", "", "r:* offline_access", "code", http.StatusFound, false, false)
	getAuthorize(t, test.ClientTestRequiredParams, "nonce", "http://localhost:7777", "", "r:* offline_access", "", http.StatusFound, true, false)
	getAuthorize(t, test.ClientTestC2C, "nonce", "http://localhost:7777", "", "", "", http.StatusOK, false, true)
	getAuthorize(t, test.ClientTestC2C, "nonce", "", "", "", "", http.StatusOK, false, false)

}

func getAuthorize(t *testing.T, clientID, nonce, redirectURI, deviceID, scope, responseType string, statusCode int, containsErrorQueryParameter, consentScreenDisplayed bool) string {
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
	if responseType != "" {
		q.Add(uri.ResponseType, responseType)
	}

	u.RawQuery = q.Encode()
	getReq := test.NewRequest(http.MethodGet, config.OAUTH_SERVER_HOST, u.String(), nil).Build()
	res := test.HTTPDo(t, getReq, false)
	defer func() {
		_ = res.Body.Close()
	}()
	require.Equal(t, statusCode, res.StatusCode)
	if res.StatusCode == http.StatusFound {
		loc, err := res.Location()
		require.NoError(t, err)
		if containsErrorQueryParameter {
			errMsg := loc.Query().Get(uri.ErrorMessageKey)
			require.NotEmpty(t, errMsg)
			return ""
		}
		code := loc.Query().Get(uri.CodeKey)
		require.NotEmpty(t, code)
		return code
	}
	if res.StatusCode == http.StatusOK {
		if consentScreenDisplayed {
			buf, err := ioutil.ReadAll(res.Body)
			require.NoError(t, err)
			require.Contains(t, string(buf), "<title>Consent Screen</title>")
			return ""
		}
		var body map[string]string
		err := json.ReadFrom(res.Body, &body)
		require.NoError(t, err)

		code := body[uri.CodeKey]
		require.NotEmpty(t, code)
		return code
	}

	return ""
}
