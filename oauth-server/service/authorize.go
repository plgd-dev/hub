package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"

	"github.com/plgd-dev/cloud/oauth-server/uri"
)

type authorizedSession struct {
	cfg   *Client
	nonce string
}

func (requestHandler *RequestHandler) authorize(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get(uri.ClientIDQueryKey)
	clientCfg := requestHandler.config.Clients.Find(clientID)
	if clientCfg == nil {
		writeError(w, fmt.Errorf("unknown client_id(%v)", clientID), http.StatusBadRequest)
		return
	}
	if !clientCfg.AllowedGrantTypes.IsAllowed(AllowedGrantType_AUTHORIZATION_CODE) {
		writeError(w, fmt.Errorf("grant_type(%v) is not supported by client(%v)", AllowedGrantType_AUTHORIZATION_CODE, clientID), http.StatusBadRequest)
		return
	}
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	nonce := r.URL.Query().Get(uri.NonceQueryKey)
	code := hex.EncodeToString(b)
	requestHandler.cache.Set(code, authorizedSession{
		cfg:   clientCfg,
		nonce: nonce,
	}, clientCfg.AuthorizationCodeLifetime)

	u := r.URL.Query().Get(uri.RedirectURIQueryKey)
	if len(u) > 0 {
		u, err := url.Parse(string(u))
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		state := r.URL.Query().Get(uri.StateQueryKey)
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
	jsonResponseWriter(w, resp)
}
