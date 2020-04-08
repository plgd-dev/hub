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
	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/cbor"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	kitHttp "github.com/go-ocf/kit/net/http"
	"github.com/go-ocf/sdk/schema/cloud"
	"github.com/gofrs/uuid"
	cache "github.com/patrickmn/go-cache"
)

func (s *SubscribeManager) subscribeToDevice(ctx context.Context, l store.LinkedAccount, correlationID, signingSecret, deviceID string) (string, error) {
	resp, err := subscribe(ctx, "/devices/"+deviceID+"/subscriptions", correlationID, events.SubscriptionRequest{
		URL: s.eventsURL,
		EventTypes: []events.EventType{
			events.EventType_ResourcesPublished,
			events.EventType_ResourcesUnpublished,
		},
		SigningSecret: signingSecret,
	}, l)
	if err != nil {
		return "", fmt.Errorf("cannot subscribe to device %v for %v: %v", deviceID, l.ID, err)
	}
	return resp.SubscriptionId, nil
}

func cancelDeviceSubscription(ctx context.Context, l store.LinkedAccount, deviceID, subscriptionID string) error {
	err := cancelSubscription(ctx, "/devices/"+deviceID+"/subscriptions/"+subscriptionID, l)
	if err != nil {
		return fmt.Errorf("cannot cancel device subscription for %v: %v", l.ID, err)
	}
	return nil
}

func (s *SubscribeManager) updateCloudStatus(ctx context.Context, deviceID string, online bool, authContext pbCQRS.AuthorizationContext, sequence uint64) error {
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
			ContentType:       gocoap.AppOcfCbor.String(),
			CoapContentFormat: int32(gocoap.AppOcfCbor),
			Data:              data,
		},
		Status: pbRA.Status_OK,
		CommandMetadata: &pbCQRS.CommandMetadata{
			ConnectionId: OpenapiConnectorConnectionId,
			Sequence:     sequence,
		},
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

// HandleResourcesPublished publish resources to resource aggregate and subscribes to resources.
func (s *SubscribeManager) HandleResourcesPublished(ctx context.Context, d subscriptionData, header events.EventHeader, links events.ResourcesPublished) error {
	userID, err := d.linkedAccount.OriginCloud.AccessToken.GetSubject()
	if err != nil {
		return fmt.Errorf("cannot get userID: %v", err)
	}
	var errors []error
	for _, link := range links {
		endpoints := make([]*pbRA.EndpointInformation, 0, 4)
		for _, endpoint := range link.GetEndpoints() {
			endpoints = append(endpoints, &pbRA.EndpointInformation{
				Endpoint: endpoint.URI,
				Priority: int64(endpoint.Priority),
			})
		}
		href := trimDeviceIDFromHref(link.DeviceID, link.Href)
		resourceId := raCqrs.MakeResourceId(link.DeviceID, kitHttp.CanonicalHref(href))
		_, err := s.raClient.PublishResource(kitNetGrpc.CtxWithToken(ctx, d.linkedAccount.OriginCloud.AccessToken.String()), &pbRA.PublishResourceRequest{
			AuthorizationContext: &pbCQRS.AuthorizationContext{
				UserId:   userID,
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
				ConnectionId: OpenapiConnectorConnectionId,
				Sequence:     header.SequenceNumber,
			},
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot publish resource: %v", err))
			continue
		}

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
			LinkedAccountID: d.linkedAccount.ID,
			DeviceID:        link.DeviceID,
			Href:            href,
			SigningSecret:   signingSecret,
		}
		err = s.cache.Add(correlationID.String(), subscriptionData{
			linkedAccount: d.linkedAccount,
			subscription:  sub,
		}, cache.DefaultExpiration)
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot cache subscription for device subscriptions: %v", err))
			continue
		}
		sub.SubscriptionID, err = s.subscribeToResource(ctx, d.linkedAccount, correlationID.String(), signingSecret, link.DeviceID, href)
		if err != nil {
			s.cache.Delete(correlationID.String())
			errors = append(errors, fmt.Errorf("cannot subscribe to device %v resource %v: %v", link.DeviceID, href, err))
			continue
		}
		_, err = s.store.FindOrCreateSubscription(ctx, sub)
		if err != nil {
			cancelResourceSubscription(ctx, d.linkedAccount, sub.DeviceID, sub.Href, sub.SubscriptionID)
			errors = append(errors, fmt.Errorf("cannot store resource subscription to DB: %v", err))
			continue
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

// HandleResourcesUnpublished unpublish resources from resource aggregate and cancel resources subscriptions.
func (s *SubscribeManager) HandleResourcesUnpublished(ctx context.Context, d subscriptionData, header events.EventHeader, links events.ResourcesUnpublished) error {
	userID, err := d.linkedAccount.OriginCloud.AccessToken.GetSubject()
	if err != nil {
		return fmt.Errorf("cannot get userID: %v", err)
	}
	var errors []error
	for _, link := range links {
		href := trimDeviceIDFromHref(link.DeviceID, link.Href)
		_, err := s.raClient.UnpublishResource(kitNetGrpc.CtxWithToken(ctx, d.linkedAccount.OriginCloud.AccessToken.String()), &pbRA.UnpublishResourceRequest{
			AuthorizationContext: &pbCQRS.AuthorizationContext{
				UserId:   userID,
				DeviceId: link.DeviceID,
			},
			ResourceId: raCqrs.MakeResourceId(link.GetDeviceID(), kitHttp.CanonicalHref(href)),
			CommandMetadata: &pbCQRS.CommandMetadata{
				ConnectionId: OpenapiConnectorConnectionId,
				Sequence:     header.SequenceNumber,
			},
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot unpublish resource: %v", err))
		}
		err = cancelResourceSubscription(ctx, d.linkedAccount, link.DeviceID, href, header.SubscriptionID)
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot unsubscribe to resource: %v", err))
		}

		err = s.store.RemoveSubscriptions(ctx, store.SubscriptionQuery{SubscriptionID: header.SubscriptionID})
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
func (s *SubscribeManager) HandleDeviceEvent(ctx context.Context, header events.EventHeader, body []byte, subscriptionData subscriptionData) error {
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
