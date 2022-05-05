package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/plgd-dev/go-coap/v2/pkg/cache"
	"github.com/plgd-dev/go-coap/v2/pkg/runner/periodic"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	kitHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	"github.com/plgd-dev/kit/v2/codec/json"
	"go.opentelemetry.io/otel/trace"
)

const AuthorizationHeader string = "Authorization"
const AcceptHeader string = "Accept"

type Type string

const (
	Type_Devices  Type = "devices"
	Type_Device   Type = "device"
	Type_Resource Type = "resource"
)

type Subscription struct {
	ID              string
	Type            Type
	LinkedAccountID string
	LinkedCloudID   string
	DeviceID        string
	Href            string
	SigningSecret   string
	CorrelationID   string
}

type SubscriptionManager struct {
	eventsURL           string
	store               *Store
	raClient            raService.ResourceAggregateClient
	isClient            pbIS.IdentityStoreClient
	cache               *cache.Cache
	devicesSubscription *DevicesSubscription
	provider            *oauth2.PlgdProvider
	triggerTask         OnTaskTrigger
	tracerProvider      trace.TracerProvider
}

func NewSubscriptionManager(
	eventsURL string,
	isClient pbIS.IdentityStoreClient,
	raClient raService.ResourceAggregateClient,
	store *Store,
	devicesSubscription *DevicesSubscription,
	provider *oauth2.PlgdProvider,
	triggerTask OnTaskTrigger,
	tracerProvider trace.TracerProvider,
) *SubscriptionManager {
	cache := cache.NewCache()
	add := periodic.New(devicesSubscription.ctx.Done(), time.Minute*5)
	add(func(now time.Time) bool {
		cache.CheckExpirations(now)
		return true
	})

	return &SubscriptionManager{
		eventsURL:           eventsURL,
		store:               store,
		raClient:            raClient,
		isClient:            isClient,
		devicesSubscription: devicesSubscription,
		cache:               cache,
		provider:            provider,
		triggerTask:         triggerTask,
		tracerProvider:      tracerProvider,
	}
}

func subscribe(ctx context.Context, tracerProvider trace.TracerProvider, href, correlationID string, reqBody events.SubscriptionRequest, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) (resp events.SubscriptionResponse, err error) {
	client := linkedCloud.GetHTTPClient(tracerProvider)
	defer client.CloseIdleConnections()

	r, w := io.Pipe()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, linkedCloud.Endpoint.URL+kitHttp.CanonicalHref(href), r)
	if err != nil {
		return resp, fmt.Errorf("cannot create post request: %w", err)
	}
	req.Header.Set(events.CorrelationIDKey, correlationID)
	req.Header.Set("Accept", events.ContentType_JSON+","+events.ContentType_VNDOCFCBOR)
	req.Header.Set(events.ContentTypeKey, events.ContentType_JSON)
	req.Header.Set(AuthorizationHeader, "Bearer "+string(linkedAccount.Data.Target().AccessToken))
	req.Header.Set("Connection", "close")
	req.Close = true

	go func() {
		defer func() {
			if err := w.Close(); err != nil {
				log.Errorf("failed to close write pipe: %v", err)
			}
		}()
		err := json.WriteTo(w, reqBody)
		if err != nil {
			log.Errorf("cannot encode %+v to json: %w", reqBody, err)
		}
	}()
	httpResp, err := client.Do(req)
	if err != nil {
		return resp, fmt.Errorf("cannot post: %w", err)
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			log.Errorf("failed to close response body stream: %w", err)
		}
	}()
	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		return resp, fmt.Errorf("unexpected statusCode %v", httpResp.StatusCode)
	}
	err = json.ReadFrom(httpResp.Body, &resp)
	if err != nil {
		return resp, fmt.Errorf("cannot device response: %w", err)
	}
	return resp, nil
}

func cancelSubscription(ctx context.Context, tracerProvider trace.TracerProvider, href string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	client := linkedCloud.GetHTTPClient(tracerProvider)
	defer client.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, linkedCloud.Endpoint.URL+kitHttp.CanonicalHref(href), nil)
	if err != nil {
		return fmt.Errorf("cannot create delete request: %w", err)
	}
	req.Header.Set("Token", linkedAccount.ID)
	req.Header.Set("Accept", events.ContentType_JSON+","+events.ContentType_VNDOCFCBOR)
	req.Header.Set(AuthorizationHeader, "Bearer "+string(linkedAccount.Data.Target().AccessToken))
	req.Header.Set("Connection", "close")
	req.Close = true

	httpResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot delete: %w", err)
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			log.Errorf("failed to close response body stream: %w", err)
		}
	}()
	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected statusCode %v", httpResp.StatusCode)
	}
	return nil
}

func (s *SubscriptionManager) getSubscriptionData(subscriptionID, correlationID string) (subscriptionData, error) {
	data := s.cache.Load(correlationID)
	if data != nil {
		subData := data.Data().(subscriptionData)
		subData.subscription.ID = subscriptionID
		newSubscription, loaded, err := s.store.LoadOrCreateSubscription(subData.subscription)
		s.cache.Delete(correlationID)
		if err != nil {
			return subscriptionData{}, fmt.Errorf("cannot store subscription(CorrelationID: %v, ID: %v) to DB: %w", correlationID, subscriptionID, err)
		}
		if loaded && newSubscription.subscription.ID != subscriptionID {
			return subscriptionData{}, fmt.Errorf("cannot store subscription(CorrelationID: %v, ID: %v) to DB: duplicit subscription(CorrelationID %v, ID: %v)",
				subscriptionID, subscriptionID, newSubscription.subscription.CorrelationID, newSubscription.subscription.ID)
		}
		subData.subscription = newSubscription.subscription
		return subData, nil
	}
	newSubscription, ok := s.store.LoadSubscription(subscriptionID)
	if !ok {
		return subscriptionData{}, fmt.Errorf("cannot load subscription(CorrelationID: %v, ID: %v) from DB: not found",
			correlationID, subscriptionID)
	}
	return newSubscription, nil
}

func (s *SubscriptionManager) HandleEvent(ctx context.Context, header events.EventHeader, body []byte) (int, error) {
	subData, err := s.getSubscriptionData(header.ID, header.CorrelationID)
	if err != nil {
		return http.StatusGone, err
	}

	// verify event signature
	calcEventSignature := events.CalculateEventSignature(subData.subscription.SigningSecret,
		header.ContentType,
		header.EventType,
		header.ID,
		header.SequenceNumber,
		header.EventTimestamp, body)
	if header.EventSignature != calcEventSignature {
		return http.StatusBadRequest, fmt.Errorf("invalid event signature %v(%+v != %+v, %s): not match", header.ID, subData.subscription, header, body)
	}

	subData.linkedAccount, err = refreshTokens(ctx, s.tracerProvider, subData.linkedAccount, subData.linkedCloud, s.provider, s.store)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("cannot refresh token: %w", err)
	}

	ctx = kitNetGrpc.CtxWithToken(ctx, subData.linkedAccount.Data.Origin().AccessToken.String())
	if header.EventType == events.EventType_SubscriptionCanceled {
		err := s.HandleCancelEvent(ctx, header, subData.linkedAccount)
		if err != nil {
			return http.StatusGone, fmt.Errorf("cannot cancel subscription: %w", err)
		}
		return http.StatusOK, nil
	}

	switch subData.subscription.Type {
	case Type_Devices:
		err = s.HandleDevicesEvent(ctx, header, body, subData)
	case Type_Device:
		err = s.HandleDeviceEvent(ctx, header, body, subData)
	case Type_Resource:
		err = s.HandleResourceEvent(ctx, header, body, subData)
	default:
		return http.StatusBadRequest, fmt.Errorf("cannot handle event %v: handler not found", header.EventType)
	}
	if err != nil {
		return http.StatusBadRequest, err
	}
	return http.StatusOK, nil
}

type LinkedAccountHandler struct {
	linkedAccounts map[string]store.LinkedAccount
}

func (h *LinkedAccountHandler) Handle(ctx context.Context, iter store.LinkedAccountIter) (err error) {
	for {
		var s store.LinkedAccount
		if !iter.Next(ctx, &s) {
			break
		}
		if h.linkedAccounts == nil {
			h.linkedAccounts = make(map[string]store.LinkedAccount)
		}
		h.linkedAccounts[s.ID] = s
	}
	return iter.Err()
}

func (s *SubscriptionManager) HandleCancelEvent(ctx context.Context, header events.EventHeader, linkedAccount store.LinkedAccount) error {
	_, ok := s.store.PullOutSubscription(header.ID)
	if !ok {
		return fmt.Errorf("cannot cancel subscription %v: not found", header.ID)
	}
	return nil
}

type subscriptionData struct {
	linkedAccount store.LinkedAccount
	linkedCloud   store.LinkedCloud
	subscription  Subscription
}

func (s *SubscriptionManager) Run(ctx context.Context, interval time.Duration) {
	for {
		for _, task := range s.store.DumpTasks() {
			s.triggerTask(task)
		}
		for _, data := range s.store.DumpDevices() {
			err := s.devicesSubscription.Add(data.subscription.DeviceID, data.linkedAccount, data.linkedCloud)
			if err != nil {
				log.Errorf("cannot add device %v from subscriptions to devicesSubscription: %w", data.subscription.DeviceID, err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}
