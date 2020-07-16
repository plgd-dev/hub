package service

import (
	"fmt"
	"net/http"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/cloud/cloud2cloud-gateway/store"

	"github.com/go-ocf/kit/log"

	"github.com/gorilla/mux"
)

func makeDevicesRepresentation(deviceID string) interface{} {
	return []map[string]string{{"di": deviceID}}
}

func (rh *RequestHandler) subscribeToDevice(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	deviceID := routeVars[deviceIDKey]

	_, userID, err := parseAuth(r.Header.Get("Authorization"))
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot parse authorization header: %w", err)
	}

	s, code, err := rh.makeSubscription(w, r, store.Type_Device, userID, []events.EventType{
		events.EventType_ResourcesPublished,
		events.EventType_ResourcesUnpublished,
	})
	if err != nil {
		return code, err
	}
	s.DeviceID = deviceID
	err = rh.subMgr.Store(r.Context(), s)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot store subscription: %w", err)
	}
	err = jsonResponseWriterEncoder(w, SubscriptionResponse{
		SubscriptionID: s.ID,
	}, http.StatusCreated)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot write response: %w", err)
	}
	err = rh.subMgr.Connect(s.ID)
	if err != nil {
		log.Errorf("cannot store subscription: %v", err)
	}
	return http.StatusOK, nil
}

func (rh *RequestHandler) SubscribeToDevice(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.subscribeToDevice(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot subscribe to device: %w", err), statusCode, w)
	}
}
