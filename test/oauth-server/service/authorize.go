package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
	"github.com/plgd-dev/kit/codec/json"
)

type authorizedSession struct {
	nonce    string
	audience string
	deviceID string
}

func (requestHandler *RequestHandler) authorize(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get(uri.ClientIDKey)
	clientCfg := clients.Find(clientID)
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
	requestHandler.cache.Set(code, authorizedSession{
		nonce:    nonce,
		audience: audience,
		deviceID: deviceId,
	}, clientCfg.AuthorizationCodeLifetime)
	responseMode := r.URL.Query().Get(uri.ResponseMode)
	state := r.URL.Query().Get(uri.StateKey)
	switch responseMode {
	case "web_message":
		v := map[string]string{
			"code":  code,
			"state": state,
		}
		code, err := json.Encode(v)
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
						var targetOrigin = "` + requestHandler.getDomain() + `";
						var authorizationResponse = {type: "authorization_response",response: ` + string(code) + `};
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
		return
	}
	u := r.URL.Query().Get(uri.RedirectURIKey)
	if len(u) > 0 {
		u, err := url.Parse(string(u))
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
		http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
		return
	}

	resp := map[string]interface{}{
		"code": code,
	}

	if err = jsonResponseWriter(w, resp); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}
