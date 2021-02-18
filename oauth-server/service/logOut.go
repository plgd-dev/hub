package service

import (
	"net/http"

	"github.com/plgd-dev/cloud/oauth-server/uri"
)

func (requestHandler *RequestHandler) logOut(w http.ResponseWriter, r *http.Request) {
	returnTo := r.URL.Query().Get(uri.ReturnToQueryKey)
	http.Redirect(w, r, returnTo, http.StatusTemporaryRedirect)
}
