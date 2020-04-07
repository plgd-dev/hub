package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/cloud/openapi-connector/events"
	"github.com/gorilla/mux"
)

func (rh *RequestHandler) unsubscribeFromDevices(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	subscriptionID := routeVars[subscriptionIDKey]
	userDevices, err := rh.GetUsersDevices(r.Context(), r)
	if err != nil {
		return http.StatusUnauthorized, err
	}
	if len(userDevices) == 0 {
		return http.StatusForbidden, fmt.Errorf("cannot get user devices: empty")
	}

	sub, err := rh.store.PopDevicesSubscription(r.Context(), subscriptionID)
	if err != nil {
		return http.StatusBadRequest, err
	}

	_, err = emitEvent(r.Context(), events.EventType_SubscriptionCanceled, sub.Subscription, func(ctx context.Context, subscriptionID string) (uint64, error) {
		return sub.SequenceNumber, nil
	}, nil)
	if err != nil {
		log.Errorf("cannot emit event: %v", err)
	}

	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot write response: %w", err)
	}

	return http.StatusOK, nil
}

func (rh *RequestHandler) UnsubscribeFromDevices(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.unsubscribeFromDevices(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot unsubscribe from all devices: %w", err), statusCode, w)
	}
}
