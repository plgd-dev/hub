package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-ocf/ocf-cloud/openapi-gateway/store"

	"github.com/gorilla/mux"
)

type retrieveDeviceSubscriptionHandler struct {
	s store.Subscription
}

func (c *retrieveDeviceSubscriptionHandler) Handle(ctx context.Context, iter store.SubscriptionIter) error {
	for iter.Next(ctx, &c.s) {
		return nil
	}
	return fmt.Errorf("not found")
}

func (rh *RequestHandler) retrieveDeviceSubscription(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	deviceID := routeVars[deviceIDKey]
	subscriptionID := routeVars[subscriptionIDKey]
	err := rh.IsAuthorized(r.Context(), r, deviceID)
	if err != nil {
		return http.StatusUnauthorized, err
	}

	res := retrieveDeviceSubscriptionHandler{}
	err = rh.store.LoadSubscriptions(r.Context(), store.SubscriptionQuery{SubscriptionID: subscriptionID}, &res)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot load subscription %v: %w", subscriptionID, err)
	}

	err = jsonResponseWriterEncoder(w, SubscriptionResponse{
		SubscriptionID: subscriptionID,
	})
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot write response: %w", err)
	}

	return http.StatusOK, nil
}

func (rh *RequestHandler) RetrieveDeviceSubscription(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.retrieveDeviceSubscription(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve device subscription: %w", err), statusCode, w)
	}
}
