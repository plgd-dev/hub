package service

import (
	"context"
	"fmt"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	raCqrs "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/go-coap/v2/message"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	kitHttp "github.com/go-ocf/kit/net/http"
)

func (s *SubscribeManager) subscribeToResource(ctx context.Context, l store.LinkedAccount, correlationID, signingSecret, deviceID, resourceHrefLink string) (string, error) {
	resp, err := subscribe(ctx, "/devices/"+deviceID+"/"+resourceHrefLink+"/subscriptions", correlationID, events.SubscriptionRequest{
		URL:           s.eventsURL,
		EventTypes:    []events.EventType{events.EventType_ResourceChanged},
		SigningSecret: signingSecret,
	}, l)
	if err != nil {
		return "", fmt.Errorf("cannot subscribe to device %v for %v: %v", deviceID, l.ID, err)
	}
	return resp.SubscriptionId, nil
}

func cancelResourceSubscription(ctx context.Context, l store.LinkedAccount, deviceID, resourceID, subscriptionID string) error {
	err := cancelSubscription(ctx, "/devices/"+deviceID+"/"+resourceID+"/subscriptions/"+subscriptionID, l)
	if err != nil {
		return fmt.Errorf("cannot cancel resource subscription for %v: %v", l.ID, err)
	}
	return nil
}

func (s *SubscribeManager) HandleResourceChangedEvent(ctx context.Context, subscriptionData subscriptionData, header events.EventHeader, body []byte) error {
	coapContentFormat := int32(-1)
	switch header.ContentType {
	case message.AppCBOR.String():
		coapContentFormat = int32(message.AppCBOR)
	case message.AppOcfCbor.String():
		coapContentFormat = int32(message.AppOcfCbor)
	case message.AppJSON.String():
		coapContentFormat = int32(message.AppJSON)
	}

	_, err := s.raClient.NotifyResourceChanged(kitNetGrpc.CtxWithToken(ctx, subscriptionData.linkedAccount.OriginCloud.AccessToken.String()), &pbRA.NotifyResourceChangedRequest{
		AuthorizationContext: &pbCQRS.AuthorizationContext{
			DeviceId: subscriptionData.subscription.DeviceID,
		},
		ResourceId: raCqrs.MakeResourceId(subscriptionData.subscription.DeviceID, kitHttp.CanonicalHref(subscriptionData.subscription.Href)),
		CommandMetadata: &pbCQRS.CommandMetadata{
			ConnectionId: Cloud2cloudConnectorConnectionId,
			Sequence:     header.SequenceNumber,
		},
		Content: &pbRA.Content{
			Data:              body,
			ContentType:       header.ContentType,
			CoapContentFormat: coapContentFormat,
		},
	})
	if err != nil {
		return fmt.Errorf("cannot update resource aggregate (%v) resource (%v) content changed: %v", subscriptionData.subscription.DeviceID, subscriptionData.subscription.Href, err)
	}

	return nil

}

func (s *SubscribeManager) HandleResourceEvent(ctx context.Context, header events.EventHeader, body []byte, subscriptionData subscriptionData) error {
	switch header.EventType {
	case events.EventType_ResourceChanged:
		return s.HandleResourceChangedEvent(ctx, subscriptionData, header, body)
	}
	return fmt.Errorf("cannot handle resource event: unsupported Event-Type %v", header.EventType)
}
