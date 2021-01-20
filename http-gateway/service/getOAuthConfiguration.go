package service

import (
	"net/http"
)

func (requestHandler *RequestHandler) getOAuthConfiguration(w http.ResponseWriter, r *http.Request) {
	jsonResponseWriter(w, requestHandler.config.UI.OAuthClient)
}
