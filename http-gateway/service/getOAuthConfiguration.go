package service

import (
	"net/http"

	"github.com/plgd-dev/cloud/pkg/log"
)

func (requestHandler *RequestHandler) getOAuthConfiguration(w http.ResponseWriter, r *http.Request) {
	if err := jsonResponseWriter(w, requestHandler.config.UI.OAuthClient); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}
