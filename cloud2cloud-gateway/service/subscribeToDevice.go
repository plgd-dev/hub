package service

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/store"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

func (rh *RequestHandler) subscribeToDevice(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	deviceID := routeVars[deviceIDKey]

	s, code, err := rh.makeSubscription(w, r, store.Type_Device, []events.EventType{
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
