package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-ocf/cloud/cloud2cloud-gateway/store"

	"github.com/gorilla/mux"
)

type retrieveResourceSubscriptionHandler struct {
	s store.Subscription
}

func (c *retrieveResourceSubscriptionHandler) Handle(ctx context.Context, iter store.SubscriptionIter) error {
	for iter.Next(ctx, &c.s) {
		return nil
	}
	return fmt.Errorf("not found")
}

func (rh *RequestHandler) retrieveResourceSubscription(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	deviceID := routeVars[deviceIDKey]
	subscriptionID := routeVars[subscriptionIDKey]
	err := rh.IsAuthorized(r.Context(), r, deviceID)
	if err != nil {
		return http.StatusUnauthorized, err
	}

	res := retrieveResourceSubscriptionHandler{}
	err = rh.store.LoadSubscriptions(r.Context(), store.SubscriptionQuery{SubscriptionID: subscriptionID}, &res)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot load subscription %v: %w", subscriptionID, err)
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
	statusCode, err := rh.retrieveResourceSubscription(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource subscription: %w", err), statusCode, w)
	}
}
