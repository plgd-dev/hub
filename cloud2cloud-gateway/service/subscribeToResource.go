package service

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/store"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/kit/v2/codec/json"
)

func (rh *RequestHandler) makeSubscription(r *http.Request, typ store.Type, validEventTypes []events.EventType) (store.Subscription, int, error) {
	var res store.Subscription
	var req events.SubscriptionRequest
	err := json.ReadFrom(r.Body, &req)
	if err != nil {
		return res, http.StatusBadRequest, fmt.Errorf("cannot decode request body: %w", err)
	}

	_, err = url.Parse(req.EventsURL)
	if err != nil {
		return res, http.StatusBadRequest, fmt.Errorf("invalid eventsurl(%w)", err)
	}

	token, err := pkgHttp.GetToken(r.Header.Get("Authorization"))
	if err != nil {
		return res, http.StatusUnauthorized, fmt.Errorf("invalid accessToken(%w)", err)
	}

	eventTypes := make([]events.EventType, 0, 10)
	for _, r := range req.EventTypes {
		for _, v := range validEventTypes {
			if r == v {
				eventTypes = append(eventTypes, r)
			}
		}
	}
	if len(eventTypes) == 0 {
		return res, http.StatusBadRequest, fmt.Errorf("invalid eventtypes(%w)", err)
	}
	res.ID = uuid.Must(uuid.NewRandom()).String()
	res.EventTypes = eventTypes
	res.URL = req.EventsURL
	res.CorrelationID = r.Header.Get(events.CorrelationIDKey)
	res.Accept = strings.Split(r.Header.Get(events.AcceptKey), ",")
	res.SigningSecret = req.SigningSecret
	res.Type = typ
	res.AccessToken = token

	return res, http.StatusOK, nil
}

func (rh *RequestHandler) subscribeToResource(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	deviceID := routeVars[deviceIDKey]
	href := routeVars[hrefKey]

	s, code, err := rh.makeSubscription(r, store.Type_Resource, []events.EventType{events.EventType_ResourceChanged})
	if err != nil {
		return code, err
	}
	s.DeviceID = deviceID
	s.Href = href

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

func (rh *RequestHandler) SubscribeToResource(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.subscribeToResource(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot subscribe to resource: %w", err), statusCode, w)
	}
}
