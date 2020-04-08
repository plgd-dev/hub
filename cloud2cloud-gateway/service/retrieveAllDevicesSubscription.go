package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-ocf/cloud/cloud2cloud-gateway/store"

	"github.com/gorilla/mux"
)

type retrieveDevicesSubscriptionHandler struct {
	s store.DevicesSubscription
}

func (c *retrieveDevicesSubscriptionHandler) Handle(ctx context.Context, iter store.DevicesSubscriptionIter) error {
	for iter.Next(ctx, &c.s) {
		return nil
	}
	return fmt.Errorf("not found")
}

func (rh *RequestHandler) retrieveDevicesSubscription(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	subscriptionID := routeVars[subscriptionIDKey]
	userDevices, err := rh.GetUsersDevices(r.Context(), r)
	if err != nil {
		return http.StatusUnauthorized, err
	}
	if len(userDevices) == 0 {
		return http.StatusForbidden, fmt.Errorf("cannot get user devices: empty")
	}

	res := retrieveDevicesSubscriptionHandler{}
	err = rh.store.LoadDevicesSubscriptions(r.Context(), store.DevicesSubscriptionQuery{SubscriptionID: subscriptionID}, &res)
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

func (rh *RequestHandler) RetrieveDevicesSubscription(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.retrieveDevicesSubscription(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve all devices subscription: %w", err), statusCode, w)
	}
}
