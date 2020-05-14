package service

import (
	"fmt"
	"net/http"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	oapiStore "github.com/go-ocf/cloud/cloud2cloud-connector/store"

	"github.com/gorilla/mux"
)

func makeOnlineOfflineRepresentation(deviceID string) interface{} {
	return []map[string]string{{"di": deviceID}}
}

func (rh *RequestHandler) subscribeToDevice(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	deviceID := routeVars[deviceIDKey]

	err := rh.IsAuthorized(r.Context(), r, deviceID)
	if err != nil {
		return http.StatusUnauthorized, err
	}

	_, userID, err := parseAuth(r.Header.Get("Authorization"))
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot parse authorization header: %w", err)
	}

	s, code, err := rh.makeSubscription(w, r, oapiStore.Type_Device, userID, []events.EventType{
		events.EventType_ResourcesPublished,
		events.EventType_ResourcesUnpublished,
	})
	if err != nil {
		return code, err
	}
	s.DeviceID = deviceID

	_, err = rh.resourceProjection.Register(r.Context(), deviceID)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot register to resource projection: %w", err)
	}
	models := rh.resourceProjection.Models(deviceID, "")
	if len(models) == 0 {
		err = rh.resourceProjection.ForceUpdate(r.Context(), deviceID, "")
		if err != nil {
			rh.resourceProjection.Unregister(deviceID)
			return http.StatusBadRequest, fmt.Errorf("cannot load resources for device: %w", err)
		}
	}

	err = rh.store.SaveSubscription(r.Context(), s)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot save subscription: %w", err)
	}

	models = rh.resourceProjection.Models(deviceID, "")
	if len(models) == 0 {
		rh.resourceProjection.Unregister(deviceID)
		rh.store.PopSubscription(r.Context(), s.ID)
		return http.StatusBadRequest, fmt.Errorf("cannot load resources for device and device: %w", err)
	}

	for _, eventType := range s.EventTypes {
		var rep interface{}
		switch eventType {
		case events.EventType_ResourcesPublished, events.EventType_ResourcesUnpublished:
			rep = makeLinksRepresentation(eventType, models)
		}

		_, err = emitEvent(r.Context(), eventType, s, rh.store.IncrementSubscriptionSequenceNumber, rep)
		if err != nil {
			rh.resourceProjection.Unregister(deviceID)
			rh.store.PopSubscription(r.Context(), s.ID)
			return http.StatusBadRequest, fmt.Errorf("cannot emit event: %w", err)
		}
	}

	err = jsonResponseWriterEncoder(w, SubscriptionResponse{
		SubscriptionID: s.ID,
	})
	if err != nil {
		rh.resourceProjection.Unregister(deviceID)
		rh.store.PopSubscription(r.Context(), s.ID)
		return http.StatusBadRequest, fmt.Errorf("cannot write response: %w", err)
	}

	return http.StatusOK, nil
}

func (rh *RequestHandler) SubscribeToDevice(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.subscribeToDevice(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot subscribe to device: %w", err), statusCode, w)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}
