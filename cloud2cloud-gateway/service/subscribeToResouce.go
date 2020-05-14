package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	oapiStore "github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"github.com/go-ocf/cloud/cloud2cloud-gateway/store"
	"github.com/go-ocf/kit/codec/json"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/gofrs/uuid"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	cqrsRA "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/gorilla/mux"
)

func (rh *RequestHandler) IsAuthorized(ctx context.Context, r *http.Request, deviceID string) error {
	token, err := getAccessToken(r)
	if err != nil {
		return fmt.Errorf("cannot authorized: cannot get users devices: %w", err)
	}

	getUserDevicesClient, err := rh.asClient.GetUserDevices(kitNetGrpc.CtxWithToken(ctx, token), &pbAS.GetUserDevicesRequest{
		DeviceIdsFilter: []string{deviceID},
	})
	if err != nil {
		return fmt.Errorf("cannot authorized: cannot get users devices: %w", err)
	}
	defer getUserDevicesClient.CloseSend()
	for {
		userDevice, err := getUserDevicesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("cannot authorized: cannot get users devices: %w", err)
		}
		if userDevice.DeviceId == deviceID {
			return nil
		}
	}
	return fmt.Errorf("cannot authorized: access denied")
}

type SubscriptionResponse struct {
	SubscriptionID string `json:"subscriptionId"`
}

func (rh *RequestHandler) makeSubscription(w http.ResponseWriter, r *http.Request, typ oapiStore.Type, userID string, validEventTypes []events.EventType) (store.Subscription, int, error) {
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
	res.ContentType = r.Header.Get(events.ContentTypeKey)
	res.UserID = userID
	res.SigningSecret = req.SigningSecret
	res.Type = typ

	return res, http.StatusOK, nil

}

func (rh *RequestHandler) subscribeToResource(w http.ResponseWriter, r *http.Request) (int, error) {
	routeVars := mux.Vars(r)
	deviceID := routeVars[deviceIDKey]
	href := routeVars[resourceLinkHrefKey]

	err := rh.IsAuthorized(r.Context(), r, deviceID)
	if err != nil {
		return http.StatusUnauthorized, err
	}

	_, userID, err := parseAuth(r.Header.Get("Authorization"))
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot parse authorization header: %w", err)
	}

	s, code, err := rh.makeSubscription(w, r, oapiStore.Type_Resource, userID, []events.EventType{events.EventType_ResourceChanged})
	if err != nil {
		return code, err
	}

	_, err = rh.resourceProjection.Register(r.Context(), deviceID)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot register to resource projection: %w", err)
	}
	resourceID := cqrsRA.MakeResourceId(deviceID, href)
	models := rh.resourceProjection.Models(deviceID, resourceID)
	if len(models) == 0 {
		err = rh.resourceProjection.ForceUpdate(r.Context(), deviceID, resourceID)
		if err != nil {
			rh.resourceProjection.Unregister(deviceID)
			return http.StatusBadRequest, fmt.Errorf("cannot load resource: %w", err)
		}
	}

	s.DeviceID = deviceID
	s.Href = href

	err = rh.store.SaveSubscription(r.Context(), s)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot save subscription: %w", err)
	}

	models = rh.resourceProjection.Models(deviceID, resourceID)
	for _, m := range models {
		resourceCtx := m.(*resourceCtx).Clone()
		if resourceCtx.content.GetStatus() != pbRA.Status_OK && resourceCtx.content.GetStatus() != pbRA.Status_UNKNOWN {
			rh.store.PopSubscription(r.Context(), s.ID)
			rh.resourceProjection.Unregister(deviceID)
			return statusToHttpStatus(resourceCtx.content.GetStatus()), fmt.Errorf("cannot prepare content to emit first event: %w", err)
		}
		rep, err := unmarshalContent(resourceCtx.content.GetContent())
		if err != nil {
			rh.store.PopSubscription(r.Context(), s.ID)
			rh.resourceProjection.Unregister(deviceID)
			return http.StatusBadRequest, fmt.Errorf("cannot prepare content to emit first event: %w", err)
		}
		_, err = emitEvent(r.Context(), events.EventType_ResourceChanged, s, rh.store.IncrementSubscriptionSequenceNumber, rep)
		if err != nil {
			rh.store.PopSubscription(r.Context(), s.ID)
			rh.resourceProjection.Unregister(deviceID)
			return http.StatusBadRequest, fmt.Errorf("cannot emit event: %w", err)
		}
	}

	err = jsonResponseWriterEncoder(w, SubscriptionResponse{
		SubscriptionID: s.ID,
	}, http.StatusCreated)
	if err != nil {
		rh.store.PopSubscription(r.Context(), s.ID)
		rh.resourceProjection.Unregister(deviceID)
		return http.StatusBadRequest, fmt.Errorf("cannot write response: %w", err)
	}

	return http.StatusOK, nil
}

func (rh *RequestHandler) SubscribeToResource(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.subscribeToResource(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot subscribe to resource: %w", err), statusCode, w)
	}
}
