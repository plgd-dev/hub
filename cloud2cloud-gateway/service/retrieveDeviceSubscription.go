package service

import (
	"fmt"
	"net/http"
)

func (rh *RequestHandler) RetrieveDeviceSubscription(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.retrieveSubscription(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve device subscription: %w", err), statusCode, w)
	}
}
