package service

import (
	"fmt"
	"net/http"

	"github.com/plgd-dev/kit/log"
	"github.com/gorilla/mux"
)

func (rh *RequestHandler) unsubscribe(w http.ResponseWriter, r *http.Request) (int, error) {
	_, userID, err := parseAuth(r.Header.Get("Authorization"))
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot parse authorization header: %w", err)
	}
	routeVars := mux.Vars(r)
	subscriptionID := routeVars[subscriptionIDKey]

	sub, err := rh.subMgr.PullOut(r.Context(), subscriptionID, userID)
	if err != nil {
		return http.StatusBadRequest, err
	}
	w.WriteHeader(http.StatusAccepted)

	err = cancelSubscription(r.Context(), rh.emitEvent, sub)
	if err != nil {
		log.Errorf("cannot emit event: %v", err)
	}

	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot write response: %w", err)
	}

	return http.StatusOK, nil
}

func (rh *RequestHandler) UnsubscribeFromDevice(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.unsubscribe(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot unsubscribe from device: %w", err), statusCode, w)
	}
}
