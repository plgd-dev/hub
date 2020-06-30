package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	raCqrs "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/kit/codec/cbor"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	kitHttp "github.com/go-ocf/kit/net/http"
	"github.com/go-ocf/sdk/schema/cloud"
	"github.com/gofrs/uuid"
	cache "github.com/patrickmn/go-cache"
)

func (s *SubscriptionManager) subscribeToDevice(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, correlationID, signingSecret, deviceID string) (string, error) {
	resp, err := subscribe(ctx, "/devices/"+deviceID+"/subscriptions", correlationID, events.SubscriptionRequest{
		URL: s.eventsURL,
		EventTypes: []events.EventType{
			events.EventType_ResourcesPublished,
			events.EventType_ResourcesUnpublished,
		},
		SigningSecret: signingSecret,
	}, linkedAccount, linkedCloud)
	if err != nil {
		return "", fmt.Errorf("cannot subscribe to device %v for %v: %v", deviceID, linkedAccount.ID, err)
	}
	return resp.SubscriptionId, nil
}

func cancelDeviceSubscription(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, deviceID, subscriptionID string) error {
	err := cancelSubscription(ctx, "/devices/"+deviceID+"/subscriptions/"+subscriptionID, linkedAccount, linkedCloud)
	if err != nil {
		return fmt.Errorf("cannot cancel device subscription for %v: %v", linkedAccount.ID, err)
	}
	return nil
}

func (s *SubscriptionManager) updateCloudStatus(ctx context.Context, deviceID string, online bool, authContext pbCQRS.AuthorizationContext, cmdMetadata pbCQRS.CommandMetadata) error {
	status := cloud.Status{
		ResourceTypes: cloud.StatusResourceTypes,
		Interfaces:    cloud.StatusInterfaces,
		Online:        online,
	}
	data, err := cbor.Encode(status)
	if err != nil {
		return err
	}

	request := pbRA.NotifyResourceChangedRequest{
		ResourceId: raCqrs.MakeResourceId(deviceID, cloud.StatusHref),
		Content: &pbRA.Content{
			ContentType:       message.AppOcfCbor.String(),
			CoapContentFormat: int32(message.AppOcfCbor),
			Data:              data,
		},
		Status:               pbRA.Status_OK,
		CommandMetadata:      &cmdMetadata,
		AuthorizationContext: &authContext,
	}

	_, err = s.raClient.NotifyResourceChanged(ctx, &request)
	return err
}

func trimDeviceIDFromHref(deviceID, href string) string {
	if strings.HasPrefix(href, "/"+deviceID+"/") {
		href = strings.TrimPrefix(href, "/"+deviceID)
	}
	return href
}

func (s *SubscriptionManager) SubscribeToResource(ctx context.Context, deviceID, href string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	signingSecret, err := generateRandomString(32)
	if err != nil {
		return fmt.Errorf("cannot generate signingSecret for device subscription: %v", err)
	}
	correlationID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("cannot generate correlationID for device subscription: %v", err)
	}

	sub := store.Subscription{
		Type:            store.Type_Resource,
		LinkedAccountID: linkedAccount.ID,
		DeviceID:        deviceID,
		Href:            href,
		SigningSecret:   signingSecret,
	}
	err = s.cache.Add(correlationID.String(), subscriptionData{
		linkedAccount: linkedAccount,
		linkedCloud:   linkedCloud,
		subscription:  sub,
	}, cache.DefaultExpiration)
	if err != nil {
		return fmt.Errorf("cannot cache subscription for device subscriptions: %v", err)
	}
	sub.ID, err = s.subscribeToResource(ctx, linkedAccount, linkedCloud, correlationID.String(), signingSecret, deviceID, href)
	if err != nil {
		s.cache.Delete(correlationID.String())
		return fmt.Errorf("cannot subscribe to device %v resource %v: %v", deviceID, href, err)
	}
	_, err = s.store.FindOrCreateSubscription(ctx, sub)
	if err != nil {
		cancelResourceSubscription(ctx, linkedAccount, linkedCloud, sub.DeviceID, sub.Href, sub.ID)
		return fmt.Errorf("cannot store resource subscription to DB: %v", err)
	}
	return nil
}

// HandleResourcesPublished publish resources to resource aggregate and subscribes to resources.
func (s *SubscriptionManager) HandleResourcesPublished(ctx context.Context, d subscriptionData, header events.EventHeader, links events.ResourcesPublished) error {
	var errors []error
	for _, link := range links {
		link.DeviceID = d.subscription.DeviceID
		endpoints := make([]*pbRA.EndpointInformation, 0, 4)
		for _, endpoint := range link.GetEndpoints() {
			endpoints = append(endpoints, &pbRA.EndpointInformation{
				Endpoint: endpoint.URI,
				Priority: int64(endpoint.Priority),
			})
		}
		href := trimDeviceIDFromHref(link.DeviceID, link.Href)
		resourceId := raCqrs.MakeResourceId(link.DeviceID, kitHttp.CanonicalHref(href))
		_, err := s.raClient.PublishResource(kitNetGrpc.CtxWithToken(ctx, d.linkedAccount.TargetCloud.AccessToken.String()), &pbRA.PublishResourceRequest{
			AuthorizationContext: &pbCQRS.AuthorizationContext{
				DeviceId: link.DeviceID,
			},
			ResourceId: resourceId,
			Resource: &pbRA.Resource{
				Id:                    resourceId,
				Href:                  href,
				ResourceTypes:         link.ResourceTypes,
				Interfaces:            link.Interfaces,
				DeviceId:              link.DeviceID,
				InstanceId:            link.InstanceID,
				Anchor:                link.Anchor,
				Policies:              &pbRA.Policies{BitFlags: int32(link.Policy.BitMask)},
				Title:                 link.Title,
				SupportedContentTypes: link.SupportedContentTypes,
				EndpointInformations:  endpoints,
			},
			CommandMetadata: &pbCQRS.CommandMetadata{
				ConnectionId: d.linkedAccount.ID + "." + d.subscription.ID,
				Sequence:     header.SequenceNumber,
			},
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot publish resource: %v", err))
			continue
		}
		if d.linkedCloud.SupportedSubscriptionsEvents.NeedPullResources() {
			continue
		}
		err = s.SubscribeToResource(ctx, link.DeviceID, href, d.linkedAccount, d.linkedCloud)
		if err != nil {
			errors = append(errors, err)
			continue
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

// HandleResourcesUnpublished unpublish resources from resource aggregate and cancel resources subscriptions.
func (s *SubscriptionManager) HandleResourcesUnpublished(ctx context.Context, d subscriptionData, header events.EventHeader, links events.ResourcesUnpublished) error {
	var errors []error
	for _, link := range links {
		link.DeviceID = d.subscription.DeviceID
		href := trimDeviceIDFromHref(link.DeviceID, link.Href)
		_, err := s.raClient.UnpublishResource(kitNetGrpc.CtxWithToken(ctx, d.linkedAccount.TargetCloud.AccessToken.String()), &pbRA.UnpublishResourceRequest{
			AuthorizationContext: &pbCQRS.AuthorizationContext{
				DeviceId: link.DeviceID,
			},
			ResourceId: raCqrs.MakeResourceId(link.GetDeviceID(), kitHttp.CanonicalHref(href)),
			CommandMetadata: &pbCQRS.CommandMetadata{
				ConnectionId: d.linkedAccount.ID + "." + d.subscription.ID,
				Sequence:     header.SequenceNumber,
			},
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot unpublish resource: %v", err))
		}
		err = s.store.RemoveSubscriptions(ctx, store.SubscriptionQuery{LinkedAccountID: d.linkedAccount.ID, DeviceID: link.DeviceID, Href: href})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot remove device %v resource %v: %v", link.DeviceID, href, err))
		}
		s.cache.Delete(header.CorrelationID)
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

// HandleDeviceEvent handles device events.
func (s *SubscriptionManager) HandleDeviceEvent(ctx context.Context, header events.EventHeader, body []byte, subscriptionData subscriptionData) error {
	contentReader, err := header.GetContentDecoder()
	if err != nil {
		return fmt.Errorf("cannot get content reader: %v", err)
	}
	switch header.EventType {
	case events.EventType_ResourcesPublished:
		var links events.ResourcesPublished
		err := contentReader(body, &links)
		if err != nil {
			return fmt.Errorf("cannot decode device event %v: %v", header.EventType, err)
		}
		return s.HandleResourcesPublished(ctx, subscriptionData, header, links)
	case events.EventType_ResourcesUnpublished:
		var links events.ResourcesUnpublished
		err := contentReader(body, &links)
		if err != nil {
			return fmt.Errorf("cannot decode device event %v: %v", header.EventType, err)
		}
		return s.HandleResourcesUnpublished(ctx, subscriptionData, header, links)
	}

	return fmt.Errorf("cannot handle device: unsupported Event-Type %v", header.EventType)
}
