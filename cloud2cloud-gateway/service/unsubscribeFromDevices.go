package service

import (
	"fmt"
	"net/http"
)

func (rh *RequestHandler) UnsubscribeFromDevices(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.unsubscribe(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot unsubscribe from all devices: %w", err), statusCode, w)
	}
}
