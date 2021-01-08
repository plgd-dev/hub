package service

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/patrickmn/go-cache"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	pbCQRS "github.com/plgd-dev/cloud/resource-aggregate/pb"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/sdk/schema/cloud"
)

func (s *SubscriptionManager) SubscribeToDevices(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	if _, loaded := s.store.LoadDevicesSubscription(linkedAccount.LinkedCloudID, linkedAccount.ID); loaded {
		return nil
	}
	signingSecret, err := generateRandomString(32)
	if err != nil {
		return fmt.Errorf("cannot generate signingSecret for start subscriptions: %w", err)
	}
	corID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("cannot generate correlationID for start subscriptions: %w", err)
	}
	correlationID := corID.String()

	sub := Subscription{
		Type:            Type_Devices,
		LinkedAccountID: linkedAccount.ID,
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
		return fmt.Errorf("cannot cache subscription for start subscriptions: %w", err)
	}
	sub.ID, err = s.subscribeToDevices(ctx, linkedAccount, linkedCloud, correlationID, signingSecret)
	if err != nil {
		s.cache.Delete(correlationID)
		return fmt.Errorf("cannot subscribe to devices for %v: %w", linkedAccount.ID, err)
	}
	_, _, err = s.store.LoadOrCreateSubscription(sub)
	if err != nil {
		cancelDevicesSubscription(ctx, linkedAccount, linkedCloud, sub.ID)
		return fmt.Errorf("cannot store subscription to DB: %w", err)
	}
	return nil
}

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
		return fmt.Errorf("cannot cancel devices subscription for %v: %w", linkedAccount.ID, err)
	}
	return nil
}

func (s *SubscriptionManager) publishCloudDeviceStatus(ctx context.Context, deviceID string, authCtx pbCQRS.AuthorizationContext, cmdMetadata pbCQRS.CommandMetadata) error {
	resource := pbRA.Resource{
		Id:            utils.MakeResourceId(deviceID, cloud.StatusHref),
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
		ResourceId: &pbRA.ResourceId{
			DeviceId: deviceID,
			Href:     cloud.StatusHref,
		},
		Resource:        &resource,
		TimeToLive:      0,
		CommandMetadata: &cmdMetadata,
	}

	_, err := s.raClient.PublishResource(ctx, &request)
	if err != nil {
		return fmt.Errorf("cannot process command publish resource: %w", err)
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
		if d.linkedCloud.SupportedSubscriptionsEvents.StaticDeviceEvents {
			s.triggerTask(Task{
				taskType:      TaskType_PullDevice,
				linkedAccount: d.linkedAccount,
				linkedCloud:   d.linkedCloud,
				deviceID:      device.ID,
			})
			continue
		}
		if d.linkedCloud.SupportedSubscriptionsEvents.NeedPullDevice() {
			continue
		}
		s.triggerTask(Task{
			taskType:      TaskType_SubscribeToDevice,
			linkedAccount: d.linkedAccount,
			linkedCloud:   d.linkedCloud,
			deviceID:      device.ID,
		})
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
		_, ok := s.store.PullOutDevice(subscriptionData.linkedAccount.LinkedCloudID, subscriptionData.linkedAccount.ID, device.ID)
		if !ok {
			log.Debugf("HandleDevicesUnregistered: cannot remove device %v subscription: not found", device.ID)
		}
		s.cache.Delete(correlationID)
		_, err := s.asClient.RemoveDevice(ctx, &pbAS.RemoveDeviceRequest{
			DeviceId: device.ID,
			UserId:   userID,
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot remove device  %v from user: %w", device.ID, err))
		}
		err = s.devicesSubscription.Delete(userID, device.ID)
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot unregister device %v from resource projection: %w", device.ID, err))
		}

	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

// HandleDevicesOnline sets device online to resource aggregate and register device to projection.
func (s *SubscriptionManager) HandleDevicesOnline(ctx context.Context, d subscriptionData, header events.EventHeader, devices events.DevicesOnline) error {
	var errors []error
	for _, device := range devices {
		authCtx := pbCQRS.AuthorizationContext{
			DeviceId: device.ID,
		}
		err := s.publishCloudDeviceStatus(ctx, device.ID, authCtx, pbCQRS.CommandMetadata{
			ConnectionId: d.linkedAccount.ID + "." + d.subscription.ID,
			Sequence:     header.SequenceNumber,
		})
		if err != nil {
			errors = append(errors, err)
			continue
		}
		err = s.updateCloudStatus(ctx, device.ID, true, authCtx, pbCQRS.CommandMetadata{
			ConnectionId: d.linkedAccount.ID + "." + d.subscription.ID,
			Sequence:     header.SequenceNumber,
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot set device %v to online: %w", device.ID, err))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}

	return nil
}

// HandleDevicesOffline sets device off to resource aggregate and unregister device to projection.
func (s *SubscriptionManager) HandleDevicesOffline(ctx context.Context, d subscriptionData, header events.EventHeader, devices events.DevicesOffline) error {
	var errors []error
	for _, device := range devices {
		authCtx := pbCQRS.AuthorizationContext{
			DeviceId: device.ID,
		}
		err := s.publishCloudDeviceStatus(ctx, device.ID, authCtx, pbCQRS.CommandMetadata{
			ConnectionId: d.linkedAccount.ID + "." + d.subscription.ID,
			Sequence:     header.SequenceNumber,
		})
		if err != nil {
			errors = append(errors, err)
			continue
		}
		err = s.updateCloudStatus(ctx, device.ID, false, authCtx, pbCQRS.CommandMetadata{
			ConnectionId: d.linkedAccount.ID + "." + d.subscription.ID,
			Sequence:     header.SequenceNumber,
		})

		if err != nil {
			errors = append(errors, fmt.Errorf("cannot set device %v to offline: %w", device.ID, err))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}

	return nil
}

func (s *SubscriptionManager) HandleDevicesEvent(ctx context.Context, header events.EventHeader, body []byte, d subscriptionData) error {
	contentReader, err := header.GetContentDecoder()
	if err != nil {
		return fmt.Errorf("cannot handle device event: %w", err)
	}

	switch header.EventType {
	case events.EventType_DevicesRegistered:
		var devices events.DevicesRegistered
		err = contentReader(body, &devices)
		if err != nil {
			return fmt.Errorf("cannot decode devices event: %w", err)
		}
		return s.HandleDevicesRegistered(ctx, d, devices, header)
	case events.EventType_DevicesUnregistered:
		var devices events.DevicesUnregistered
		err = contentReader(body, &devices)
		if err != nil {
			return fmt.Errorf("cannot decode devices event: %w", err)
		}
		return s.HandleDevicesUnregistered(ctx, d, header.CorrelationID, devices)
	case events.EventType_DevicesOnline:
		var devices events.DevicesOnline
		err = contentReader(body, &devices)
		if err != nil {
			return fmt.Errorf("cannot decode devices event: %w", err)
		}
		return s.HandleDevicesOnline(ctx, d, header, devices)
	case events.EventType_DevicesOffline:
		var devices events.DevicesOffline
		err = contentReader(body, &devices)
		if err != nil {
			return fmt.Errorf("cannot decode devices event: %w", err)
		}
		return s.HandleDevicesOffline(ctx, d, header, devices)
	}

	return fmt.Errorf("cannot decode devices: unsupported Event-Type %v", header.EventType)
}
