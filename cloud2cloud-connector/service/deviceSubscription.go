package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	cache "github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"go.opentelemetry.io/otel/trace"
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
	data := SubscriptionData{
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
		var errors *multierror.Error
		errors = multierror.Append(errors, fmt.Errorf("cannot store subscription to DB: %w", err))
		if err2 := cancelDeviceSubscription(ctx, s.tracerProvider, linkedAccount, linkedCloud, deviceID, sub.ID); err2 != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot cancel device %v subscription: %w", deviceID, err2))
		}
		return errors.ErrorOrNil()
	}
	err = s.devicesSubscription.Add(ctx, deviceID, linkedAccount, linkedCloud)
	if err != nil {
		return fmt.Errorf("cannot register device %v to resource projection: %w", deviceID, err)
	}
	return nil
}

func (s *SubscriptionManager) subscribeToDevice(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, correlationID, signingSecret, deviceID string) (string, error) {
	resp, err := subscribe(ctx, s.tracerProvider, prefixDevicesPath+deviceID+"/subscriptions", correlationID, events.SubscriptionRequest{
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
	return resp.SubscriptionID, nil
}

func cancelDeviceSubscription(ctx context.Context, tracerProvider trace.TracerProvider, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, deviceID, subscriptionID string) error {
	err := cancelSubscription(ctx, tracerProvider, prefixDevicesPath+deviceID+"/subscriptions/"+subscriptionID, linkedAccount, linkedCloud)
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

// handleResourcesPublished publish resources to resource aggregate and subscribes to resources.
func (s *SubscriptionManager) handleResourcesPublished(ctx context.Context, d SubscriptionData, header events.EventHeader, links events.ResourcesPublished) error {
	var errors *multierror.Error
	for _, link := range links {
		deviceID := d.subscription.DeviceID
		link.DeviceID = deviceID
		endpoints := make([]*commands.EndpointInformation, 0, 4)
		for _, endpoint := range link.GetEndpoints() {
			endpoints = append(endpoints, &commands.EndpointInformation{
				Endpoint: endpoint.URI,
				Priority: endpoint.Priority,
			})
		}
		href := pkgHttpUri.CanonicalHref(trimDeviceIDFromHref(link.DeviceID, link.Href))
		_, err := s.raClient.PublishResourceLinks(ctx, &commands.PublishResourceLinksRequest{
			DeviceId: link.DeviceID,
			Resources: []*commands.Resource{{
				Href:                  href,
				ResourceTypes:         link.ResourceTypes,
				Interfaces:            link.Interfaces,
				DeviceId:              link.DeviceID,
				Anchor:                link.Anchor,
				Policy:                &commands.Policy{BitFlags: commands.ToPolicyBitFlags(link.Policy.BitMask)},
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
			errors = multierror.Append(errors, fmt.Errorf("cannot publish resource: %w", err))
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
	return errors.ErrorOrNil()
}

// handleResourcesUnpublished unpublish resources from resource aggregate and cancel resources subscriptions.
func (s *SubscriptionManager) handleResourcesUnpublished(ctx context.Context, d SubscriptionData, header events.EventHeader, links events.ResourcesUnpublished) error {
	var errors *multierror.Error
	for _, link := range links {
		link.DeviceID = d.subscription.DeviceID
		href := pkgHttpUri.CanonicalHref(trimDeviceIDFromHref(link.DeviceID, link.Href))
		_, err := s.raClient.UnpublishResourceLinks(ctx, &commands.UnpublishResourceLinksRequest{
			DeviceId: link.GetDeviceID(),
			Hrefs:    []string{href},
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: d.linkedAccount.ID + "." + d.subscription.ID,
				Sequence:     header.SequenceNumber,
			},
		})
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot unpublish resource: %w", err))
		}
		_, ok := s.store.PullOutResource(d.linkedAccount.LinkedCloudID, d.linkedAccount.ID, link.DeviceID, href)
		if !ok {
			log.Debugf("handleResourcesUnpublished: cannot remove device %v resource %v subscription: not found", link.DeviceID, href)
		}
		s.cache.Delete(header.CorrelationID)
	}
	return errors.ErrorOrNil()
}

// handleDeviceEvent handles device events.
func (s *SubscriptionManager) handleDeviceEvent(ctx context.Context, header events.EventHeader, body []byte, subscriptionData SubscriptionData) error {
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
		return s.handleResourcesPublished(ctx, subscriptionData, header, links)
	case events.EventType_ResourcesUnpublished:
		var links events.ResourcesUnpublished
		err := contentReader(body, &links)
		if err != nil {
			return fmt.Errorf("cannot decode device event %v: %w", header.EventType, err)
		}
		return s.handleResourcesUnpublished(ctx, subscriptionData, header, links)
	}

	return fmt.Errorf("cannot handle device: unsupported Event-Type %v", header.EventType)
}
