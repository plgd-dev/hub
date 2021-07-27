package service

import (
	"net/http"

	"github.com/plgd-dev/cloud/pkg/log"
)

func (requestHandler *RequestHandler) getUserInfo(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"sub": deviceUserID,
	}

	if err := jsonResponseWriter(w, resp); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}
