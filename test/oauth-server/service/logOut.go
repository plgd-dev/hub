package service

import (
	"net/http"

	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
)

func (requestHandler *RequestHandler) logOut(w http.ResponseWriter, r *http.Request) {
	redirectURI := ""
	switch r.Method {
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		redirectURI = r.Form.Get(uri.PostLogoutRedirectURIKey)
	case http.MethodGet:
		redirectURI = r.URL.Query().Get(uri.PostLogoutRedirectURIKey)
	default:
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	if redirectURI != "" {
		http.Redirect(w, r, redirectURI, http.StatusTemporaryRedirect)
		return
	}
	w.WriteHeader(http.StatusOK)
}
