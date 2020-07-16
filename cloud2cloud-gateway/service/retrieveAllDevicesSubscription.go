package service

import (
	"fmt"
	"net/http"
)

func (rh *RequestHandler) RetrieveDevicesSubscription(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.retrieveSubscription(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve all devices subscription: %w", err), statusCode, w)
	}
}
