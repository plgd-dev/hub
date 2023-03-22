package service

import (
	"net/http"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/log"
)

func (requestHandler *RequestHandler) getUserInfo(w http.ResponseWriter, _ *http.Request) {
	resp := map[string]interface{}{
		"sub": DeviceUserID,
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	if err := jsonResponseWriter(w, resp); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}
