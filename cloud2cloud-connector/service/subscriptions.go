package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
	pbIS "github.com/plgd-dev/cloud/identity-store/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	kitHttp "github.com/plgd-dev/cloud/pkg/net/http"
	"github.com/plgd-dev/cloud/pkg/security/oauth2"
	raService "github.com/plgd-dev/cloud/resource-aggregate/service"
	"github.com/plgd-dev/kit/codec/json"
	"github.com/plgd-dev/kit/log"
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
	interval            time.Duration
}

func NewSubscriptionManager(
	EventsURL string,
	isClient pbIS.IdentityStoreClient,
	raClient raService.ResourceAggregateClient,
	store *Store,
	devicesSubscription *DevicesSubscription,
	provider *oauth2.PlgdProvider,
	triggerTask OnTaskTrigger,
	interval time.Duration,
) *SubscriptionManager {
	return &SubscriptionManager{
		eventsURL:           EventsURL,
		store:               store,
		raClient:            raClient,
		isClient:            isClient,
		devicesSubscription: devicesSubscription,
		cache:               cache.New(time.Minute*10, time.Minute*5),
		provider:            provider,
		triggerTask:         triggerTask,
		interval:            interval,
	}
}

func subscribe(ctx context.Context, href, correlationID string, reqBody events.SubscriptionRequest, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) (resp events.SubscriptionResponse, err error) {
	client := linkedCloud.GetHTTPClient()
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
			log.Errorf("failed to close response body stream: %v")
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

func cancelSubscription(ctx context.Context, href string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	client := linkedCloud.GetHTTPClient()
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
			log.Errorf("failed to close response body stream: %v")
		}
	}()
	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected statusCode %v", httpResp.StatusCode)
	}
	return nil
}

func (s *SubscriptionManager) HandleEvent(ctx context.Context, header events.EventHeader, body []byte) (int, error) {
	var subData subscriptionData
	var err error
	data, ok := s.cache.Get(header.CorrelationID)
	if ok {
		subData = data.(subscriptionData)
		subData.subscription.ID = header.ID
		newSubscription, loaded, err := s.store.LoadOrCreateSubscription(subData.subscription)
		s.cache.Delete(header.CorrelationID)
		if err != nil {
			return http.StatusGone, fmt.Errorf("cannot store subscription(CorrelationID: %v, ID: %v) to DB: %w", header.CorrelationID, header.ID, err)
		}
		if loaded && newSubscription.subscription.ID != header.ID {
			return http.StatusGone, fmt.Errorf("cannot store subscription(CorrelationID: %v, ID: %v) to DB: duplicit subscription(CorrelationID %v, ID: %v)", header.CorrelationID, header.ID, newSubscription.subscription.CorrelationID, newSubscription.subscription.ID)
		}
		subData.subscription = newSubscription.subscription
	} else {
		newSubscription, ok := s.store.LoadSubscription(header.ID)
		if !ok {
			return http.StatusGone, fmt.Errorf("cannot load subscription(CorrelationID: %v, ID: %v) from DB: not found", header.CorrelationID, header.ID)
		}
		subData = newSubscription
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

	subData.linkedAccount, err = refreshTokens(ctx, subData.linkedAccount, subData.linkedCloud, s.provider, s.store)
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

func (s *SubscriptionManager) Run(ctx context.Context) {
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
		case <-time.After(s.interval):
		}
	}
}
