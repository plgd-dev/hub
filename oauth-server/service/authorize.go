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
	nonce    string
	audience string
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
	code := hex.EncodeToString(b)
	requestHandler.cache.Set(code, authorizedSession{
		nonce:    nonce,
		audience: audience,
	}, clientCfg.AuthorizationCodeLifetime)

	u := r.URL.Query().Get(uri.RedirectURIKey)
	if len(u) > 0 {
		u, err := url.Parse(string(u))
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		state := r.URL.Query().Get(uri.StateKey)
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
