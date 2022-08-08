package service

import (
	"fmt"
	"net/http"
)

func (rh *RequestHandler) UnsubscribeFromResource(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.unsubscribe(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot unsubscribe from resource: %w", err), statusCode, w)
	}
}
