package service

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/patrickmn/go-cache"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/go-coap/v2/message"
	kitHttp "github.com/plgd-dev/kit/net/http"
)

func (s *SubscriptionManager) SubscribeToResource(ctx context.Context, deviceID, href string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	if _, loaded := s.store.LoadResourceSubscription(linkedAccount.LinkedCloudID, linkedAccount.ID, deviceID, href); loaded {
		return nil
	}

	signingSecret, err := generateRandomString(32)
	if err != nil {
		return fmt.Errorf("cannot generate signingSecret for device subscription: %w", err)
	}
	corID, err := uuid.NewV4()
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
	data := subscriptionData{
		linkedAccount: linkedAccount,
		linkedCloud:   linkedCloud,
		subscription:  sub,
	}
	err = s.cache.Add(correlationID, data, cache.DefaultExpiration)
	if err != nil {
		return fmt.Errorf("cannot cache subscription for device subscriptions: %w", err)
	}
	sub.ID, err = s.subscribeToResource(ctx, linkedAccount, linkedCloud, correlationID, signingSecret, deviceID, href)
	if err != nil {
		s.cache.Delete(correlationID)
		return fmt.Errorf("cannot subscribe to device %v resource %v: %w", deviceID, href, err)
	}
	_, _, err = s.store.LoadOrCreateSubscription(sub)
	if err != nil {
		cancelResourceSubscription(ctx, linkedAccount, linkedCloud, sub.DeviceID, sub.Href, sub.ID)
		return fmt.Errorf("cannot store resource subscription to DB: %w", err)
	}
	return nil
}

func (s *SubscriptionManager) subscribeToResource(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, correlationID, signingSecret, deviceID, href string) (string, error) {
	resp, err := subscribe(ctx, "/devices/"+deviceID+href+"/subscriptions", correlationID, events.SubscriptionRequest{
		URL:           s.eventsURL,
		EventTypes:    []events.EventType{events.EventType_ResourceChanged},
		SigningSecret: signingSecret,
	}, linkedAccount, linkedCloud)
	if err != nil {
		return "", fmt.Errorf("cannot subscribe to device %v for %v: %w", deviceID, linkedAccount.ID, err)
	}
	return resp.SubscriptionId, nil
}

func cancelResourceSubscription(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, deviceID, href, subscriptionID string) error {
	err := cancelSubscription(ctx, "/devices/"+deviceID+href+"/subscriptions/"+subscriptionID, linkedAccount, linkedCloud)
	if err != nil {
		return fmt.Errorf("cannot cancel resource subscription for %v: %w", linkedAccount.ID, err)
	}
	return nil
}

func (s *SubscriptionManager) HandleResourceChangedEvent(ctx context.Context, subscriptionData subscriptionData, header events.EventHeader, body []byte) error {
	coapContentFormat := int32(-1)
	switch header.ContentType {
	case message.AppCBOR.String():
		coapContentFormat = int32(message.AppCBOR)
	case message.AppOcfCbor.String():
		coapContentFormat = int32(message.AppOcfCbor)
	case message.AppJSON.String():
		coapContentFormat = int32(message.AppJSON)
	}

	_, err := s.raClient.NotifyResourceChanged(ctx, &commands.NotifyResourceChangedRequest{
		AuthorizationContext: &commands.AuthorizationContext{
			DeviceId: subscriptionData.subscription.DeviceID,
		},
		ResourceId: commands.NewResourceID(subscriptionData.subscription.DeviceID, kitHttp.CanonicalHref(subscriptionData.subscription.Href)),
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

func (s *SubscriptionManager) HandleResourceEvent(ctx context.Context, header events.EventHeader, body []byte, subscriptionData subscriptionData) error {
	switch header.EventType {
	case events.EventType_ResourceChanged:
		return s.HandleResourceChangedEvent(ctx, subscriptionData, header, body)
	}
	return fmt.Errorf("cannot handle resource event: unsupported Event-Type %v", header.EventType)
}
