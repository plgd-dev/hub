package service

import (
	"net/http"

	"github.com/plgd-dev/hub/test/oauth-server/uri"
)

func (requestHandler *RequestHandler) logOut(w http.ResponseWriter, r *http.Request) {
	returnTo := r.URL.Query().Get(uri.ReturnToKey)
	http.Redirect(w, r, returnTo, http.StatusTemporaryRedirect)
}
