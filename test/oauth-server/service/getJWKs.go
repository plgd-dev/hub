package service

import (
	"net/http"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/plgd-dev/hub/pkg/log"
)

func (requestHandler *RequestHandler) getJWKs(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"keys": []jwk.Key{
			requestHandler.idTokenJwkKey,
			requestHandler.accessTokenJwkKey,
		},
	}

	if err := jsonResponseWriter(w, resp); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}
