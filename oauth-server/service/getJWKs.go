package service

import (
	"net/http"

	"github.com/lestrrat-go/jwx/jwk"
)

func (requestHandler *RequestHandler) getJWKs(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"keys": []jwk.Key{
			requestHandler.idTokenJwkKey,
			requestHandler.accessTokenJwkKey,
		},
	}
	jsonResponseWriter(w, resp)
}
