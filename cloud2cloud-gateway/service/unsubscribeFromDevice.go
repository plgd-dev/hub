package service

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/kit/log"
)

func (rh *RequestHandler) unsubscribe(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	subscriptionID := routeVars[subscriptionIDKey]

	sub, err := rh.subMgr.PullOut(r.Context(), subscriptionID)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = cancelSubscription(r.Context(), rh.emitEvent, sub)
	if err != nil {
		log.Errorf("cannot emit event: %w", err)
		return http.StatusBadRequest, fmt.Errorf("cannot write response: %w", err)
	}

	w.WriteHeader(http.StatusAccepted)
	return http.StatusOK, nil
}

func (rh *RequestHandler) UnsubscribeFromDevice(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.unsubscribe(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot unsubscribe from device: %w", err), statusCode, w)
	}
}
