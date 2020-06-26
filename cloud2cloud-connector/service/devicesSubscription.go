package service

import (
	"context"
	"fmt"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	raCqrs "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/sdk/schema/cloud"
	"github.com/gofrs/uuid"
	"github.com/patrickmn/go-cache"
)

func (s *SubscriptionManager) subscribeToDevices(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, correlationID, signingSecret string) (string, error) {
	resp, err := subscribe(ctx, "/devices/subscriptions", correlationID, events.SubscriptionRequest{
		URL: s.eventsURL,
		EventTypes: []events.EventType{
			events.EventType_DevicesRegistered, events.EventType_DevicesUnregistered,
			events.EventType_DevicesOnline, events.EventType_DevicesOffline,
		},
		SigningSecret: signingSecret,
	}, linkedAccount, linkedCloud)
	if err != nil {
		return "", err
	}
	return resp.SubscriptionId, nil
}

func cancelDevicesSubscription(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, subscriptionID string) error {
	err := cancelSubscription(ctx, "/devices/subscriptions/"+subscriptionID, linkedAccount, linkedCloud)
	if err != nil {
		return fmt.Errorf("cannot cancel devices subscription for %v: %v", linkedAccount.ID, err)
	}
	return nil
}

func (s *SubscriptionManager) publishCloudDeviceStatus(ctx context.Context, deviceID string, authCtx pbCQRS.AuthorizationContext, sequence uint64) error {
	resource := pbRA.Resource{
		Id:            raCqrs.MakeResourceId(deviceID, cloud.StatusHref),
		Href:          cloud.StatusHref,
		ResourceTypes: cloud.StatusResourceTypes,
		Interfaces:    cloud.StatusInterfaces,
		DeviceId:      deviceID,
		Policies: &pbRA.Policies{
			BitFlags: 3,
		},
		Title: "Cloud device status",
	}
	request := pbRA.PublishResourceRequest{
		AuthorizationContext: &authCtx,
		ResourceId:           resource.Id,
		Resource:             &resource,
		TimeToLive:           0,
		CommandMetadata: &pbCQRS.CommandMetadata{
			Sequence:     sequence,
			ConnectionId: Cloud2cloudConnectorConnectionId,
		},
	}

	_, err := s.raClient.PublishResource(ctx, &request)
	if err != nil {
		return fmt.Errorf("cannot process command publish resource: %v", err)
	}
	return nil
}

func (s *SubscriptionManager) SubscribeToDevice(ctx context.Context, deviceID string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	signingSecret, err := generateRandomString(32)
	if err != nil {
		return fmt.Errorf("cannot generate signingSecret for device subscription: %v", err)
	}
	corID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("cannot generate correlationID for devices subscription: %v", err)
	}
	correlationID := corID.String()
	sub := store.Subscription{
		Type:            store.Type_Device,
		LinkedAccountID: linkedAccount.ID,
		DeviceID:        deviceID,
		SigningSecret:   signingSecret,
	}
	err = s.cache.Add(correlationID, subscriptionData{
		linkedAccount: linkedAccount,
		linkedCloud:   linkedCloud,
		subscription:  sub,
	}, cache.DefaultExpiration)
	if err != nil {
		return fmt.Errorf("cannot cache subscription for device subscriptions: %v", err)
	}
	sub.SubscriptionID, err = s.subscribeToDevice(ctx, linkedAccount, linkedCloud, correlationID, signingSecret, deviceID)
	if err != nil {
		s.cache.Delete(correlationID)
		return fmt.Errorf("cannot subscribe to device %v: %v", deviceID, err)
	}
	_, err = s.store.FindOrCreateSubscription(ctx, sub)
	if err != nil {
		cancelDeviceSubscription(ctx, linkedAccount, linkedCloud, deviceID, sub.SubscriptionID)
		return fmt.Errorf("cannot store subscription to DB: %v", err)
	}
	err = s.devicesSubscription.Add(deviceID, linkedAccount, linkedCloud)
	if err != nil {
		return fmt.Errorf("cannot register device %v to resource projection: %v", deviceID, err)
	}
	return nil
}

func (s *SubscriptionManager) HandleDevicesRegistered(ctx context.Context, d subscriptionData, devices events.DevicesRegistered, header events.EventHeader) error {
	var errors []error
	for _, device := range devices {
		_, err := s.asClient.AddDevice(ctx, &pbAS.AddDeviceRequest{
			DeviceId: device.ID,
			UserId:   d.linkedAccount.UserID,
		})
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if d.linkedCloud.SupportedSubscriptionsEvents.NeedPullDevice() {
			continue
		}
		err = s.SubscribeToDevice(ctx, device.ID, d.linkedAccount, d.linkedCloud)
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

func (s *SubscriptionManager) HandleDevicesUnregistered(ctx context.Context, subscriptionData subscriptionData, correlationID string, devices events.DevicesUnregistered) error {
	userID := subscriptionData.linkedAccount.UserID
	var errors []error
	for _, device := range devices {
		err := s.store.RemoveSubscriptions(ctx, store.SubscriptionQuery{LinkedAccountID: subscriptionData.linkedAccount.ID, DeviceID: device.ID})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot remove device %v subscription: %v", device.ID, err))
		}
		s.cache.Delete(correlationID)
		_, err = s.asClient.RemoveDevice(ctx, &pbAS.RemoveDeviceRequest{
			DeviceId: device.ID,
			UserId:   userID,
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot remove device  %v from user: %v", device.ID, err))
		}
		err = s.devicesSubscription.Delete(userID, device.ID)
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot unregister device %v from resource projection: %v", device.ID, err))
		}

	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

// HandleDevicesOnline sets device online to resource aggregate and register device to projection.
func (s *SubscriptionManager) HandleDevicesOnline(ctx context.Context, subscriptionData subscriptionData, header events.EventHeader, devices events.DevicesOnline) error {
	var errors []error
	for _, device := range devices {
		authCtx := pbCQRS.AuthorizationContext{
			DeviceId: device.ID,
		}
		err := s.publishCloudDeviceStatus(ctx, device.ID, authCtx, header.SequenceNumber)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		err = s.updateCloudStatus(ctx, device.ID, true, authCtx, header.SequenceNumber)
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot set device %v to online: %v", device.ID, err))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}

	return nil
}

// HandleDevicesOffline sets device off to resource aggregate and unregister device to projection.
func (s *SubscriptionManager) HandleDevicesOffline(ctx context.Context, subscriptionData subscriptionData, header events.EventHeader, devices events.DevicesOffline) error {
	var errors []error
	for _, device := range devices {
		authCtx := pbCQRS.AuthorizationContext{
			DeviceId: device.ID,
		}
		err := s.publishCloudDeviceStatus(ctx, device.ID, authCtx, header.SequenceNumber)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		err = s.updateCloudStatus(ctx, device.ID, false, authCtx, header.SequenceNumber)

		if err != nil {
			errors = append(errors, fmt.Errorf("cannot set device %v to offline: %v", device.ID, err))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}

	return nil
}

func (s *SubscriptionManager) HandleDevicesEvent(ctx context.Context, header events.EventHeader, body []byte, subscriptionData subscriptionData) error {
	contentReader, err := header.GetContentDecoder()
	if err != nil {
		return fmt.Errorf("cannot handle device event: %v", err)
	}

	switch header.EventType {
	case events.EventType_DevicesRegistered:
		var devices events.DevicesRegistered
		err = contentReader(body, &devices)
		if err != nil {
			return fmt.Errorf("cannot decode devices event: %v", err)
		}
		return s.HandleDevicesRegistered(ctx, subscriptionData, devices, header)
	case events.EventType_DevicesUnregistered:
		var devices events.DevicesUnregistered
		err = contentReader(body, &devices)
		if err != nil {
			return fmt.Errorf("cannot decode devices event: %v", err)
		}
		return s.HandleDevicesUnregistered(ctx, subscriptionData, header.CorrelationID, devices)
	case events.EventType_DevicesOnline:
		var devices events.DevicesOnline
		err = contentReader(body, &devices)
		if err != nil {
			return fmt.Errorf("cannot decode devices event: %v", err)
		}
		return s.HandleDevicesOnline(ctx, subscriptionData, header, devices)
	case events.EventType_DevicesOffline:
		var devices events.DevicesOffline
		err = contentReader(body, &devices)
		if err != nil {
			return fmt.Errorf("cannot decode devices event: %v", err)
		}
		return s.HandleDevicesOffline(ctx, subscriptionData, header, devices)
	}

	return fmt.Errorf("cannot decode devices: unsupported Event-Type %v", header.EventType)
}
