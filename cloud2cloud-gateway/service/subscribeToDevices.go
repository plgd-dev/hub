package service

import (
	"fmt"
	"net/http"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/store"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

func (rh *RequestHandler) subscribeToDevices(w http.ResponseWriter, r *http.Request) (int, error) {
	s, code, err := rh.makeSubscription(w, r, store.Type_Devices, []events.EventType{
		events.EventType_DevicesRegistered,
		events.EventType_DevicesUnregistered,
		events.EventType_DevicesOnline,
		events.EventType_DevicesOffline,
	})
	if err != nil {
		return code, err
	}

	err = rh.subMgr.Store(r.Context(), s)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot store subscription: %w", err)
	}
	err = jsonResponseWriterEncoder(w, events.SubscriptionResponse{
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

func (rh *RequestHandler) SubscribeToDevices(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.subscribeToDevices(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot subscribe to all devices: %w", err), statusCode, w)
	}
}
