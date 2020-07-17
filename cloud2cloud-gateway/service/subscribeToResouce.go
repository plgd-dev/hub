package service

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"

	"github.com/go-ocf/cloud/cloud2cloud-gateway/store"
	"github.com/go-ocf/kit/codec/json"
	"github.com/go-ocf/kit/log"
	"github.com/gofrs/uuid"

	"github.com/gorilla/mux"
)

type SubscriptionResponse struct {
	SubscriptionID string `json:"subscriptionId"`
}

func (rh *RequestHandler) makeSubscription(w http.ResponseWriter, r *http.Request, typ store.Type, userID string, validEventTypes []events.EventType) (store.Subscription, int, error) {
	var res store.Subscription
	var req events.SubscriptionRequest
	err := json.ReadFrom(r.Body, &req)
	if err != nil {
		return res, http.StatusBadRequest, fmt.Errorf("cannot decode request body: %w", err)
	}

	_, err = url.Parse(req.URL)
	if err != nil {
		return res, http.StatusBadRequest, fmt.Errorf("invalid eventsurl(%v)", err)
	}

	eventTypes := make([]events.EventType, 0, 10)
	for _, r := range req.EventTypes {
		ev := events.EventType(r)
		for _, v := range validEventTypes {
			if ev == v {
				eventTypes = append(eventTypes, ev)
			}
		}
	}
	if len(eventTypes) == 0 {
		return res, http.StatusBadRequest, fmt.Errorf("invalid eventtypes(%v)", err)
	}
	res.ID = uuid.Must(uuid.NewV4()).String()
	res.EventTypes = eventTypes
	res.URL = req.URL
	res.CorrelationID = r.Header.Get(events.CorrelationIDKey)
	res.Accept = strings.Split(r.Header.Get(events.AcceptKey), ",")
	res.UserID = userID
	res.SigningSecret = req.SigningSecret
	res.Type = typ

	return res, http.StatusOK, nil

}

func (rh *RequestHandler) subscribeToResource(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	deviceID := routeVars[deviceIDKey]
	href := routeVars[HrefKey]

	_, userID, err := parseAuth(r.Header.Get("Authorization"))
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot parse authorization header: %w", err)
	}

	s, code, err := rh.makeSubscription(w, r, store.Type_Resource, userID, []events.EventType{events.EventType_ResourceChanged})
	if err != nil {
		return code, err
	}
	s.DeviceID = deviceID
	s.Href = href

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

func (rh *RequestHandler) SubscribeToResource(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.subscribeToResource(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot subscribe to resource: %w", err), statusCode, w)
	}
}
