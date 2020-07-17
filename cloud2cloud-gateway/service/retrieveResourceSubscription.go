package service

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func (rh *RequestHandler) retrieveSubscription(w http.ResponseWriter, r *http.Request) (int, error) {
	_, userID, err := parseAuth(r.Header.Get("Authorization"))
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot parse authorization header: %w", err)
	}
	routeVars := mux.Vars(r)
	subscriptionID := routeVars[subscriptionIDKey]

	_, ok := rh.subMgr.Load(subscriptionID, userID)
	if !ok {
		return http.StatusNotFound, fmt.Errorf("not found")
	}

	err = jsonResponseWriterEncoder(w, SubscriptionResponse{
		SubscriptionID: subscriptionID,
	}, http.StatusOK)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot write response: %w", err)
	}

	return http.StatusOK, nil
}

func (rh *RequestHandler) RetrieveResourceSubscription(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.retrieveSubscription(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource subscription: %w", err), statusCode, w)
	}
}
