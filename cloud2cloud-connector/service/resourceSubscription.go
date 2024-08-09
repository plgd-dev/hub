package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"go.opentelemetry.io/otel/trace"
)

func (s *SubscriptionManager) SubscribeToResource(ctx context.Context, deviceID, href string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	if _, loaded := s.store.LoadResourceSubscription(linkedAccount.LinkedCloudID, linkedAccount.ID, deviceID, href); loaded {
		return nil
	}

	signingSecret, err := generateRandomString(32)
	if err != nil {
		return fmt.Errorf("cannot generate signingSecret for device subscription: %w", err)
	}
	corID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("cannot generate correlationID for device subscription: %w", err)
	}

	correlationID := corID.String()
	sub := Subscription{
		Type:            Type_Resource,
		LinkedAccountID: linkedAccount.ID,
		DeviceID:        deviceID,
		Href:            href,
		SigningSecret:   signingSecret,
		LinkedCloudID:   linkedCloud.ID,
		CorrelationID:   correlationID,
	}
	data := SubscriptionData{
		linkedAccount: linkedAccount,
		linkedCloud:   linkedCloud,
		subscription:  sub,
	}
	_, loaded := s.cache.LoadOrStore(correlationID, cache.NewElement(data, time.Now().Add(CacheExpiration), nil))
	if loaded {
		return fmt.Errorf("cannot cache subscription for device subscriptions: subscription with %v already exists", correlationID)
	}
	sub.ID, err = s.subscribeToResource(ctx, linkedAccount, linkedCloud, correlationID, signingSecret, deviceID, href)
	if err != nil {
		s.cache.Delete(correlationID)
		return fmt.Errorf("cannot subscribe to device %v resource %v: %w", deviceID, href, err)
	}
	_, _, err = s.store.LoadOrCreateSubscription(sub)
	if err != nil {
		var errors *multierror.Error
		errors = multierror.Append(errors, fmt.Errorf("cannot store resource subscription to DB: %w", err))
		if err2 := cancelResourceSubscription(ctx, s.tracerProvider, linkedAccount, linkedCloud, sub.DeviceID, sub.Href, sub.ID); err2 != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot cancel resource /%v%v subscription: %w", sub.DeviceID, sub.Href, err2))
		}
		return errors.ErrorOrNil()
	}
	return nil
}

func (s *SubscriptionManager) subscribeToResource(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, correlationID, signingSecret, deviceID, href string) (string, error) {
	resp, err := subscribe(ctx, s.tracerProvider, prefixDevicesPath+deviceID+href+"/subscriptions", correlationID, events.SubscriptionRequest{
		EventsURL:     s.eventsURL,
		EventTypes:    []events.EventType{events.EventType_ResourceChanged},
		SigningSecret: signingSecret,
	}, linkedAccount, linkedCloud)
	if err != nil {
		return "", fmt.Errorf("cannot subscribe to device %v for %v: %w", deviceID, linkedAccount.ID, err)
	}
	return resp.SubscriptionID, nil
}

func cancelResourceSubscription(ctx context.Context, traceProvider trace.TracerProvider, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, deviceID, href, subscriptionID string) error {
	err := cancelSubscription(ctx, traceProvider, prefixDevicesPath+deviceID+href+"/subscriptions/"+subscriptionID, linkedAccount, linkedCloud)
	if err != nil {
		return fmt.Errorf("cannot cancel resource subscription for %v: %w", linkedAccount.ID, err)
	}
	return nil
}

func (s *SubscriptionManager) handleResourceChangedEvent(ctx context.Context, subscriptionData SubscriptionData, header events.EventHeader, body []byte) error {
	coapContentFormat := stringToSupportedMediaType(header.ContentType)
	_, err := s.raClient.NotifyResourceChanged(ctx, &commands.NotifyResourceChangedRequest{
		ResourceId: commands.NewResourceID(subscriptionData.subscription.DeviceID, pkgHttpUri.CanonicalHref(subscriptionData.subscription.Href)),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: subscriptionData.linkedAccount.ID + "." + subscriptionData.subscription.ID,
			Sequence:     header.SequenceNumber,
		},
		Content: &commands.Content{
			Data:              body,
			ContentType:       header.ContentType,
			CoapContentFormat: coapContentFormat,
		},
	})
	if err != nil {
		return fmt.Errorf("cannot update resource aggregate (%v) resource (%v) content changed: %w", subscriptionData.subscription.DeviceID, subscriptionData.subscription.Href, err)
	}

	return nil
}

func (s *SubscriptionManager) handleResourceEvent(ctx context.Context, header events.EventHeader, body []byte, subscriptionData SubscriptionData) error {
	if header.EventType == events.EventType_ResourceChanged {
		return s.handleResourceChangedEvent(ctx, subscriptionData, header, body)
	}
	return fmt.Errorf("cannot handle resource event: unsupported Event-Type %v", header.EventType)
}
