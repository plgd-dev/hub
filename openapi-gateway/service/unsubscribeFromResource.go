package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/ocf-cloud/openapi-connector/events"
	"github.com/gorilla/mux"
)

func (rh *RequestHandler) unsubscribeFromResource(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	deviceID := routeVars[deviceIDKey]
	subscriptionID := routeVars[subscriptionIDKey]
	err := rh.IsAuthorized(r.Context(), r, deviceID)
	if err != nil {
		return http.StatusUnauthorized, err
	}

	sub, err := rh.store.PopSubscription(r.Context(), subscriptionID)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = rh.resourceProjection.Unregister(deviceID)
	if err != nil {
		log.Errorf("cannot unregister resource projection for %v: %v", deviceID, err)
	}

	_, err = emitEvent(r.Context(), events.EventType_SubscriptionCanceled, sub, func(ctx context.Context, subscriptionID string) (uint64, error) {
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

func (rh *RequestHandler) UnsubscribeFromResource(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.unsubscribeFromResource(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot unsubscribe from resource: %w", err), statusCode, w)
	}
}
