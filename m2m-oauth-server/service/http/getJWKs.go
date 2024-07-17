package http

import (
	"net/http"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

func (requestHandler *RequestHandler) getJWKs(w http.ResponseWriter, _ *http.Request) {
	resp := map[string]interface{}{
		"keys": []jwk.Key{
			requestHandler.m2mOAuthServiceServer.GetJWK(),
		},
	}

	if err := jsonResponseWriter(w, resp); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}
