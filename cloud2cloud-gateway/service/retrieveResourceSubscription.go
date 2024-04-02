package service

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
)

func (rh *RequestHandler) retrieveSubscription(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	subscriptionID := routeVars[subscriptionIDKey]
	href := routeVars[hrefKey]

	sub, ok := rh.subMgr.Load(subscriptionID)
	if !ok {
		return http.StatusNotFound, errors.New("not found")
	}

	if href != "" && sub.Href != href {
		return http.StatusBadRequest, fmt.Errorf("invalid resource(%v) for subscription", href)
	}

	err := jsonResponseWriterEncoder(w, events.SubscriptionResponse{
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
