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
	pkgStrings "github.com/plgd-dev/hub/pkg/strings"
	"github.com/plgd-dev/hub/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
)

type authorizedSession struct {
	nonce    string
	audience string
	deviceID string
	scope    string
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

	responseType := r.URL.Query().Get(uri.ResponseTypeKey)
	responseMode := r.URL.Query().Get(uri.ResponseModeKey)
	state := r.URL.Query().Get(uri.StateKey)
	nonce := r.URL.Query().Get(uri.NonceKey)
	audience := r.URL.Query().Get(uri.AudienceKey)
	deviceId := r.URL.Query().Get(uri.DeviceIDKey)
	code := hex.EncodeToString(b)
	scope := DefaultScope
	redirectURI := r.URL.Query().Get(uri.RedirectURIKey)
	if r.URL.Query().Get(uri.ScopeKey) != "" {
		scope = strings.Join(r.URL.Query()[uri.ScopeKey], " ")
	}

	redirectURIwithErr, err := requestHandler.validateAuthorizeRequest(clientCfg, responseType, scope, redirectURI)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	if redirectURIwithErr != "" {
		http.Redirect(w, r, redirectURIwithErr, http.StatusFound)
		return
	}

	requestHandler.authSession.LoadOrStore(code, cache.NewElement(authorizedSession{
		nonce:    nonce,
		audience: audience,
		deviceID: deviceId,
		scope:    scope,
	}, time.Now().Add(clientCfg.AuthorizationCodeLifetime), nil))

	if responseMode == "web_message" {
		writeWebMessage(w, code, state, requestHandler.getDomain())
		return
	}

	if redirectURI == "" {
		// tests require returned code even with invalid redirect url
		resp := map[string]interface{}{
			uri.CodeKey: code,
		}
		if err = jsonResponseWriter(w, resp); err != nil {
			log.Errorf("failed to write response: %v", err)
		}
		return
	}
	successRedirectURI, err := buildRedirectURI(redirectURI, string(state), code, "")
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	if clientCfg.ConsentScreenEnabled {
		writeConsentScreen(w, redirectURI, scope, string(state), code)
		return
	}
	http.Redirect(w, r, successRedirectURI, http.StatusFound)
}

func (requestHandler *RequestHandler) validateAuthorizeRequest(clientCfg *Client, responseType, scope, redirectURI string) (newRedirectURI string, err error) {
	if clientCfg.RequiredRedirectURI != "" && clientCfg.RequiredRedirectURI != redirectURI {
		return "", fmt.Errorf("invalid redirect uri(%v)", redirectURI)
	}
	if clientCfg.RequiredResponseType != "" && clientCfg.RequiredResponseType != responseType {
		redirectURI, err := buildRedirectURI(redirectURI, "", "", "invalid response type")
		if err != nil {
			return "", err
		}
		return redirectURI, nil
	}
	if clientCfg.RequiredScope != nil {
		tScope := strings.Split(scope, " ")
		refScope := pkgStrings.MakeSortedSlice(clientCfg.RequiredScope)
		if !(pkgStrings.MakeSortedSlice(tScope).IsSubslice(refScope)) {
			redirectURI, err := buildRedirectURI(redirectURI, "", "", "invalid scope")
			if err != nil {
				return "", err
			}
			return redirectURI, nil
		}
	}
	return "", nil
}

func buildRedirectURI(redirectURI, state, code, errMsg string) (string, error) {
	u, err := url.Parse(string(redirectURI))
	if err != nil {
		return "", err
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", err
	}
	if state != "" {
		q.Add(uri.StateKey, string(state))
	}
	if code != "" {
		q.Add(uri.CodeKey, code)
	}
	if errMsg != "" {
		q.Add(uri.ErrorMessageKey, errMsg)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func writeWebMessage(w http.ResponseWriter, code, state, domain string) {
	v := map[string]string{
		uri.CodeKey:  code,
		uri.StateKey: state,
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

func writeConsentScreen(w http.ResponseWriter, redirectURI, scope, state, code string) {
	body := `
	<!DOCTYPE html>
	<html>
		<head>
			<title>Consent Screen</title>
		</head>
		<body>
			<center>
				</br></br></br>
				<p>Hello! The OAuth Client is requesting access to scope: <b>'` + scope + `'</b></p>
				<form action="` + redirectURI + `">
					<input type="hidden" name="state" value="` + string(state) + `" />
					<input type="hidden" name="code" value="` + code + `" />
					<input style="background-color: lime; font-size: 16px" type="submit" value="ACCEPT" />
				</form>
				</br>
				<form action="` + redirectURI + `">
					<input type="hidden" name="state" value="` + string(state) + `" />
					<input type="hidden" name="error" value="access_denied" />
					<input style="background-color: tomato; font-size: 16px" type="submit" value="DECLINE" />
				</form>
			</center>
		</body>
	</html>`

	w.WriteHeader(http.StatusOK)
	w.Header().Set(contentTypeHeaderKey, "text/html;charset=UTF-8")
	if _, err := w.Write([]byte(body)); err != nil {
		writeError(w, err, http.StatusBadRequest)
	}
}
