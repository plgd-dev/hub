package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/patrickmn/go-cache"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/kit/codec/json"
	"github.com/go-ocf/kit/log"
	kitHttp "github.com/go-ocf/kit/net/http"
	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	projectionRA "github.com/go-ocf/cloud/resource-aggregate/cqrs/projection"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
)

const AuthorizationHeader string = "Authorization"
const AcceptHeader string = "Accept"
const OpenapiConnectorConnectionId string = "cloud2cloud-connector"

type SubscribeManager struct {
	eventsURL          string
	store              store.Store
	raClient           pbRA.ResourceAggregateClient
	asClient           pbAS.AuthorizationServiceClient
	resourceProjection *projectionRA.Projection
	cache              *cache.Cache
}

func NewSubscriptionManager(EventsURL string, asClient pbAS.AuthorizationServiceClient, raClient pbRA.ResourceAggregateClient,
	store store.Store, resourceProjection *projectionRA.Projection) *SubscribeManager {
	return &SubscribeManager{
		eventsURL:          EventsURL,
		store:              store,
		raClient:           raClient,
		asClient:           asClient,
		cache:              cache.New(time.Minute*10, time.Minute*5),
		resourceProjection: resourceProjection,
	}
}

func subscribe(ctx context.Context, href, correlationID string, reqBody events.SubscriptionRequest, l store.LinkedAccount) (resp events.SubscriptionResponse, err error) {
	client := http.Client{}

	r, w := io.Pipe()

	req, err := http.NewRequest("POST", l.TargetURL+kitHttp.CanonicalHref(href), r)
	if err != nil {
		return resp, fmt.Errorf("cannot create post request: %v", err)
	}
	req.Header.Set(events.CorrelationIDKey, correlationID)
	req.Header.Set("Accept", events.ContentType_JSON+","+events.ContentType_CBOR+","+events.ContentType_VNDOCFCBOR)
	req.Header.Set(events.ContentTypeKey, events.ContentType_JSON)
	req.Header.Set(AuthorizationHeader, "Bearer "+string(l.TargetCloud.AccessToken))

	go func() {
		defer w.Close()
		err := json.WriteTo(w, reqBody)
		if err != nil {
			log.Errorf("cannot encode to json: %v", err)
		}
	}()
	httpResp, err := client.Do(req)
	if err != nil {
		return resp, fmt.Errorf("cannot post: %v", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("unexpected statusCode %v", httpResp.StatusCode)
	}
	err = json.ReadFrom(httpResp.Body, &resp)
	if err != nil {
		return resp, fmt.Errorf("cannot device response: %v", err)
	}
	return resp, nil
}

func cancelSubscription(ctx context.Context, href string, l store.LinkedAccount) error {
	client := http.Client{}
	req, err := http.NewRequest("DELETE", l.TargetURL+kitHttp.CanonicalHref(href), nil)
	if err != nil {
		return fmt.Errorf("cannot create delete request: %v", err)
	}
	req.Header.Set("Token", l.ID)
	req.Header.Set("Accept", events.ContentType_JSON+","+events.ContentType_CBOR+","+events.ContentType_VNDOCFCBOR)
	req.Header.Set(AuthorizationHeader, "Bearer "+string(l.TargetCloud.AccessToken))

	httpResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot delete: %v", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected statusCode %v", httpResp.StatusCode)
	}
	return nil
}

type SubscriptionHandler struct {
	subscription store.Subscription
	ok           bool
}

func (h *SubscriptionHandler) Handle(ctx context.Context, iter store.SubscriptionIter) (err error) {
	var s store.Subscription
	if iter.Next(ctx, &s) {
		h.ok = true
		h.subscription = s
		return iter.Err()
	}
	return iter.Err()
}

func (s *SubscribeManager) HandleEvent(ctx context.Context, header events.EventHeader, body []byte) (int, error) {
	var subData subscriptionData
	var err error
	data, ok := s.cache.Get(header.CorrelationID)
	if ok {
		subData = data.(subscriptionData)
		subData.subscription.SubscriptionID = header.SubscriptionID
		newSubscription, err := s.store.FindOrCreateSubscription(ctx, subData.subscription)
		if err != nil {
			cancelDevicesSubscription(ctx, subData.linkedAccount, subData.subscription.SubscriptionID)
			return http.StatusGone, fmt.Errorf("cannot store subscription to DB: %v", err)
		}
		subData.subscription = newSubscription
	} else {
		var h SubscriptionHandler
		err := s.store.LoadSubscriptions(ctx, []store.SubscriptionQuery{store.SubscriptionQuery{SubscriptionID: header.SubscriptionID}}, &h)
		if err != nil {
			return http.StatusGone, fmt.Errorf("cannot load subscription from DB: %v", err)
		}
		if !h.ok {
			return http.StatusGone, fmt.Errorf("unknown subscription %v, eventType %v", header.SubscriptionID, header.EventType)
		}
		subData.subscription = h.subscription
		var lh LinkedAccountHandler
		err = s.store.LoadLinkedAccounts(ctx, store.Query{ID: subData.subscription.LinkedAccountID}, &lh)
		if err != nil {
			return http.StatusGone, fmt.Errorf("cannot load linked account for subscription %v: %v", header.SubscriptionID, err)
		}
		if !h.ok {
			return http.StatusGone, fmt.Errorf("unknown linked account %v subscription %v", subData.subscription.LinkedAccountID, subData.subscription.SubscriptionID)
		}
		subData.linkedAccount = lh.linkedAccount
	}

	// verify event signature
	if header.EventSignature != events.CalculateEventSignature(subData.subscription.SigningSecret,
		header.ContentType,
		header.EventType,
		header.SubscriptionID,
		header.SequenceNumber,
		header.EventTimestamp, body) {
		return http.StatusBadRequest, fmt.Errorf("invalid event signature %v: %v", header.SubscriptionID, err)
	}

	s.cache.Set(header.CorrelationID, subData, cache.DefaultExpiration)

	subData.linkedAccount, err = subData.linkedAccount.RefreshTokens(ctx, s.store)
	if err != nil {
		return http.StatusGone, fmt.Errorf("cannot refresh access token for linked account %v: %v", subData.linkedAccount.ID, err)
	}

	if header.EventType == events.EventType_SubscriptionCanceled {
		err := s.HandleCancelEvent(ctx, header, subData.linkedAccount)
		if err != nil {
			return http.StatusGone, fmt.Errorf("cannot cancel subscription: %v", err)
		}
		return http.StatusOK, nil
	}

	switch subData.subscription.Type {
	case store.Type_Devices:
		err = s.HandleDevicesEvent(ctx, header, body, subData)

	case store.Type_Device:
		err = s.HandleDeviceEvent(ctx, header, body, subData)
	case store.Type_Resource:
		err = s.HandleResourceEvent(ctx, header, body, subData)
	default:
		return http.StatusGone, fmt.Errorf("cannot handle event %v: handler not found", header.EventType)
	}
	if err != nil {
		return http.StatusGone, err
	}
	return http.StatusOK, nil
}

type LinkedAccountHandler struct {
	linkedAccount store.LinkedAccount
	ok            bool
}

func (h *LinkedAccountHandler) Handle(ctx context.Context, iter store.LinkedAccountIter) (err error) {
	var s store.LinkedAccount
	if iter.Next(ctx, &s) {
		h.ok = true
		h.linkedAccount = s
		return iter.Err()
	}
	return fmt.Errorf("not found")
}

func (s *SubscribeManager) HandleCancelEvent(ctx context.Context, header events.EventHeader, linkedAccount store.LinkedAccount) error {
	var h SubscriptionHandler
	err := s.store.LoadSubscriptions(ctx, []store.SubscriptionQuery{store.SubscriptionQuery{SubscriptionID: header.SubscriptionID}}, &h)
	if err != nil {
		return fmt.Errorf("cannot load subscription from DB: %v", err)
	}
	if !h.ok {
		return fmt.Errorf("unknown subscription %v, eventType %v", header.SubscriptionID, header.EventType)
	}
	return s.store.RemoveSubscriptions(ctx, store.SubscriptionQuery{SubscriptionID: header.SubscriptionID})
}

type subscriptionData struct {
	linkedAccount store.LinkedAccount
	subscription  store.Subscription
}

func (s *SubscribeManager) StartSubscriptions(ctx context.Context, l store.LinkedAccount) error {
	signingSecret, err := generateRandomString(32)
	if err != nil {
		return fmt.Errorf("cannot generate signingSecret for start subscriptions: %v", err)
	}
	corID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("cannot generate correlationID for start subscriptions: %v", err)
	}
	correlationID := corID.String()

	sub := store.Subscription{
		Type:            store.Type_Devices,
		LinkedAccountID: l.ID,
		SigningSecret:   signingSecret,
	}
	err = s.cache.Add(correlationID, subscriptionData{
		linkedAccount: l,
		subscription:  sub,
	}, cache.DefaultExpiration)
	if err != nil {
		return fmt.Errorf("cannot cache subscription for start subscriptions: %v", err)
	}
	sub.SubscriptionID, err = s.subscribeToDevices(ctx, l, correlationID, signingSecret)
	if err != nil {
		s.cache.Delete(correlationID)
		return fmt.Errorf("cannot subscribe to devices for %v: %v", l.ID, err)
	}
	_, err = s.store.FindOrCreateSubscription(ctx, sub)
	if err != nil {
		cancelDevicesSubscription(ctx, l, sub.SubscriptionID)
		return fmt.Errorf("cannot store subscription to DB: %v", err)
	}
	return nil
}

type SubscriptionsHandler struct {
	subscriptions []store.Subscription
}

func (h *SubscriptionsHandler) Handle(ctx context.Context, iter store.SubscriptionIter) (err error) {
	var s store.Subscription
	for iter.Next(ctx, &s) {
		h.subscriptions = append(h.subscriptions, s)
	}
	return iter.Err()
}

func (s *SubscribeManager) StopSubscriptions(ctx context.Context, l store.LinkedAccount) error {
	var h SubscriptionsHandler
	err := s.store.LoadSubscriptions(ctx, []store.SubscriptionQuery{store.SubscriptionQuery{LinkedAccountID: l.ID}}, &h)
	if err != nil {
		return fmt.Errorf("cannot load subscriptions: %v", err)
	}
	if len(h.subscriptions) == 0 {
		return nil
	}
	linkedAccount, err := l.RefreshTokens(ctx, s.store)

	var errors []error
	for _, sub := range h.subscriptions {
		switch sub.Type {
		case store.Type_Devices:
			err = cancelDevicesSubscription(ctx, linkedAccount, sub.SubscriptionID)
			if err != nil {
				errors = append(errors, err)
			}
		case store.Type_Device:
			err = cancelDeviceSubscription(ctx, linkedAccount, sub.DeviceID, sub.SubscriptionID)
			if err != nil {
				errors = append(errors, err)
			}
		case store.Type_Resource:
			err = cancelResourceSubscription(ctx, linkedAccount, sub.DeviceID, sub.Href, sub.SubscriptionID)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	err = s.store.RemoveSubscriptions(ctx, store.SubscriptionQuery{LinkedAccountID: l.ID})
	if err != nil {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}

	return nil
}
