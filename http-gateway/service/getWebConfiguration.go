package service

import (
	"net/http"

	"github.com/plgd-dev/hub/v2/pkg/log"
)

func (requestHandler *RequestHandler) getWebConfiguration(w http.ResponseWriter, r *http.Request) {
	if err := jsonResponseWriter(w, requestHandler.config.UI.WebConfiguration); err != nil {
		log.Errorf("failed to write response: %w", err)
	}
}