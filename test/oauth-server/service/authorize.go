package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/plgd-dev/go-coap/v2/pkg/cache"
	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/hub/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
)

type authorizedSession struct {
	nonce    string
	audience string
	deviceID string
	scopes   string
}

func (requestHandler *RequestHandler) authorize(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get(uri.ClientIDKey)
	clientCfg := requestHandler.config.OAuthSigner.Clients.Find(clientID)
	if clientCfg == nil {
		writeError(w, fmt.Errorf("unknown client_id(%v)", clientID), http.StatusBadRequest)
		return
	}
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	nonce := r.URL.Query().Get(uri.NonceKey)
	audience := r.URL.Query().Get(uri.AudienceKey)
	deviceId := r.URL.Query().Get(uri.DeviceId)
	code := hex.EncodeToString(b)
	scopes := "openid profile email"
	if r.URL.Query().Get(uri.ScopeKey) != "" {
		scopes = strings.Join(r.URL.Query()[uri.ScopeKey], " ")
	}
	requestHandler.authSession.LoadOrStore(code, cache.NewElement(authorizedSession{
		nonce:    nonce,
		audience: audience,
		deviceID: deviceId,
		scopes:   scopes,
	}, time.Now().Add(clientCfg.AuthorizationCodeLifetime), nil))
	responseMode := r.URL.Query().Get(uri.ResponseMode)
	state := r.URL.Query().Get(uri.StateKey)

	if responseMode == "web_message" {
		writeWebMessage(w, code, state, requestHandler.getDomain())
		return
	}

	// redirect url flow
	ru := r.URL.Query().Get(uri.RedirectURIKey)
	if ru == "" {
		// tests require returned code even with invalid redirect url
		resp := map[string]interface{}{
			"code": code,
		}
		if err = jsonResponseWriter(w, resp); err != nil {
			log.Errorf("failed to write response: %v", err)
		}
		return
	}

	u, err := url.Parse(string(ru))
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	q.Add("state", string(state))
	q.Add("code", code)
	u.RawQuery = q.Encode()

	if clientCfg.ConsentScreenEnabled {
		writeConsentScreen(w, scopes, u)
	}
	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}

func writeWebMessage(w http.ResponseWriter, code, state, domain string) {
	v := map[string]string{
		"code":  code,
		"state": state,
	}
	json, err := json.Encode(v)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	body := `
	<!DOCTYPE html>
	<html>
		<head>
			<title>Authorization Response</title>
		</head>
		<body>
			<script type="text/javascript">
				(function(window, document) {
					var targetOrigin = "` + domain + `";
					var authorizationResponse = {type: "authorization_response",response: ` + string(json) + `};
					var mainWin = (window.opener) ? window.opener : window.parent;
					mainWin.postMessage(authorizationResponse, targetOrigin);
				})(this, this.document);
			</script>
		</body>
	</html>`
	w.WriteHeader(http.StatusOK)
	w.Header().Set(contentTypeHeaderKey, "text/html;charset=UTF-8")
	if _, err = w.Write([]byte(body)); err != nil {
		log.Errorf("failed to write response body: %v", err)
	}
}

func writeConsentScreen(w http.ResponseWriter, scopes string, redirectUrl *url.URL) {
	body := `
	<!DOCTYPE html>
	<html>
		<head>
			<title>Conset Screen</title>
		</head>
		<body>
			<center>
				</br></br></br>
				<p>Hello! The OAuth Client is requesting access to scopes: <b>'` + scopes + `'</b></p>
				<form action="` + redirectUrl.String() + `">
					<input style="background-color: lime; font-size: 16px" type="submit" value="ACCEPT" />
				</form>
				</br>
				<form action="` + uri.UnauthorizedResponse + `">
					<input style="background-color: tomato; font-size: 16px" type="submit" value="DECLINE" />
				</form>
			</center>
		</body>
	</html>`

	w.WriteHeader(http.StatusOK)
	w.Header().Set(contentTypeHeaderKey, "text/html;charset=UTF-8")
	if _, err := w.Write([]byte(body)); err != nil {
		log.Errorf("failed to write response body: %v", err)
	}
}
