package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	cache "github.com/plgd-dev/go-coap/v2/pkg/cache"
	"github.com/plgd-dev/hub/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/cloud2cloud-connector/store"
	kitHttp "github.com/plgd-dev/hub/pkg/net/http"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/kit/v2/log"
)

func (s *SubscriptionManager) SubscribeToDevice(ctx context.Context, deviceID string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	if _, loaded := s.store.LoadDeviceSubscription(linkedAccount.LinkedCloudID, linkedAccount.ID, deviceID); loaded {
		return nil
	}
	signingSecret, err := generateRandomString(32)
	if err != nil {
		return fmt.Errorf("cannot generate signingSecret for device subscription: %w", err)
	}
	corID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("cannot generate correlationID for devices subscription: %w", err)
	}
	correlationID := corID.String()
	sub := Subscription{
		Type:            Type_Device,
		LinkedAccountID: linkedAccount.ID,
		DeviceID:        deviceID,
		SigningSecret:   signingSecret,
		LinkedCloudID:   linkedCloud.ID,
		CorrelationID:   correlationID,
	}
	data := subscriptionData{
		linkedAccount: linkedAccount,
		linkedCloud:   linkedCloud,
		subscription:  sub,
	}
	_, loaded := s.cache.LoadOrStore(correlationID, cache.NewElement(data, time.Now().Add(CacheExpiration), nil))
	if loaded {
		return fmt.Errorf("cannot cache subscription for device subscriptions: subscription with %v already exists", correlationID)
	}
	sub.ID, err = s.subscribeToDevice(ctx, linkedAccount, linkedCloud, correlationID, signingSecret, deviceID)
	if err != nil {
		s.cache.Delete(correlationID)
		return fmt.Errorf("cannot subscribe to device %v: %w", deviceID, err)
	}
	_, _, err = s.store.LoadOrCreateSubscription(sub)
	if err != nil {
		var errors []error = make([]error, 1, 2)
		errors = append(errors, fmt.Errorf("cannot store subscription to DB: %w", err))
		if err2 := cancelDeviceSubscription(ctx, linkedAccount, linkedCloud, deviceID, sub.ID); err2 != nil {
			errors = append(errors, fmt.Errorf("cannot cancel device %v subscription: %w", deviceID, err2))
		}
		return fmt.Errorf("%v", errors)
	}
	err = s.devicesSubscription.Add(deviceID, linkedAccount, linkedCloud)
	if err != nil {
		return fmt.Errorf("cannot register device %v to resource projection: %w", deviceID, err)
	}
	return nil
}

func (s *SubscriptionManager) subscribeToDevice(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, correlationID, signingSecret, deviceID string) (string, error) {
	resp, err := subscribe(ctx, "/devices/"+deviceID+"/subscriptions", correlationID, events.SubscriptionRequest{
		EventsURL: s.eventsURL,
		EventTypes: []events.EventType{
			events.EventType_ResourcesPublished,
			events.EventType_ResourcesUnpublished,
		},
		SigningSecret: signingSecret,
	}, linkedAccount, linkedCloud)
	if err != nil {
		return "", fmt.Errorf("cannot subscribe to device %v for %v: %w", deviceID, linkedAccount.ID, err)
	}
	return resp.SubscriptionId, nil
}

func cancelDeviceSubscription(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, deviceID, subscriptionID string) error {
	err := cancelSubscription(ctx, "/devices/"+deviceID+"/subscriptions/"+subscriptionID, linkedAccount, linkedCloud)
	if err != nil {
		return fmt.Errorf("cannot cancel device subscription for %v: %w", linkedAccount.ID, err)
	}
	return nil
}

func trimDeviceIDFromHref(deviceID, href string) string {
	if strings.HasPrefix(href, "/"+deviceID+"/") {
		href = strings.TrimPrefix(href, "/"+deviceID)
	}
	return href
}

// HandleResourcesPublished publish resources to resource aggregate and subscribes to resources.
func (s *SubscriptionManager) HandleResourcesPublished(ctx context.Context, d subscriptionData, header events.EventHeader, links events.ResourcesPublished) error {
	var errors []error
	for _, link := range links {
		deviceID := d.subscription.DeviceID
		link.DeviceID = deviceID
		endpoints := make([]*commands.EndpointInformation, 0, 4)
		for _, endpoint := range link.GetEndpoints() {
			endpoints = append(endpoints, &commands.EndpointInformation{
				Endpoint: endpoint.URI,
				Priority: int64(endpoint.Priority),
			})
		}
		href := kitHttp.CanonicalHref(trimDeviceIDFromHref(link.DeviceID, link.Href))
		_, err := s.raClient.PublishResourceLinks(ctx, &commands.PublishResourceLinksRequest{
			DeviceId: link.DeviceID,
			Resources: []*commands.Resource{{
				Href:                  href,
				ResourceTypes:         link.ResourceTypes,
				Interfaces:            link.Interfaces,
				DeviceId:              link.DeviceID,
				Anchor:                link.Anchor,
				Policy:                &commands.Policy{BitFlags: int32(link.Policy.BitMask)},
				Title:                 link.Title,
				SupportedContentTypes: link.SupportedContentTypes,
				EndpointInformations:  endpoints,
			}},
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: d.linkedAccount.ID + "." + d.subscription.ID,
				Sequence:     header.SequenceNumber,
			},
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot publish resource: %w", err))
			continue
		}
		if d.linkedCloud.SupportedSubscriptionEvents.NeedPullResources() {
			continue
		}
		s.triggerTask(Task{
			taskType:      TaskType_SubscribeToResource,
			linkedAccount: d.linkedAccount,
			linkedCloud:   d.linkedCloud,
			deviceID:      deviceID,
			href:          href,
		})
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
		href := kitHttp.CanonicalHref(trimDeviceIDFromHref(link.DeviceID, link.Href))
		_, err := s.raClient.UnpublishResourceLinks(ctx, &commands.UnpublishResourceLinksRequest{
			DeviceId: link.GetDeviceID(),
			Hrefs:    []string{href},
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: d.linkedAccount.ID + "." + d.subscription.ID,
				Sequence:     header.SequenceNumber,
			},
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot unpublish resource: %w", err))
		}
		_, ok := s.store.PullOutResource(d.linkedAccount.LinkedCloudID, d.linkedAccount.ID, link.DeviceID, href)
		if !ok {
			log.Debugf("HandleResourcesUnpublished: cannot remove device %v resource %v subscription: not found", link.DeviceID, href)
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
		return fmt.Errorf("cannot get content reader: %w", err)
	}
	switch header.EventType {
	case events.EventType_ResourcesPublished:
		var links events.ResourcesPublished
		err := contentReader(body, &links)
		if err != nil {
			return fmt.Errorf("cannot decode device event %v: %w", header.EventType, err)
		}
		return s.HandleResourcesPublished(ctx, subscriptionData, header, links)
	case events.EventType_ResourcesUnpublished:
		var links events.ResourcesUnpublished
		err := contentReader(body, &links)
		if err != nil {
			return fmt.Errorf("cannot decode device event %v: %w", header.EventType, err)
		}
		return s.HandleResourcesUnpublished(ctx, subscriptionData, header, links)
	}

	return fmt.Errorf("cannot handle device: unsupported Event-Type %v", header.EventType)
}
