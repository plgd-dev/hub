package service

import (
	"context"
	"fmt"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/cloud/openapi-connector/events"
	"github.com/go-ocf/cloud/openapi-connector/store"
	raCqrs "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/sdk/schema/cloud"
	"github.com/gofrs/uuid"
	"github.com/patrickmn/go-cache"
)

func (s *SubscribeManager) subscribeToDevices(ctx context.Context, l store.LinkedAccount, correlationID, signingSecret string) (string, error) {

	resp, err := subscribe(ctx, "/devices/subscriptions", correlationID, events.SubscriptionRequest{
		URL: s.eventsURL,
		EventTypes: []events.EventType{
			events.EventType_DevicesRegistered, events.EventType_DevicesUnregistered,
			events.EventType_DevicesOnline, events.EventType_DevicesOffline,
		},
		SigningSecret: signingSecret,
	}, l)
	if err != nil {
		return "", err
	}
	return resp.SubscriptionId, nil
}

func cancelDevicesSubscription(ctx context.Context, l store.LinkedAccount, subscriptionID string) error {
	err := cancelSubscription(ctx, "/devices/subscriptions/"+subscriptionID, l)
	if err != nil {
		return fmt.Errorf("cannot cancel devices subscription for %v: %v", l.ID, err)
	}
	return nil
}

func (s *SubscribeManager) publishCloudDeviceStatus(ctx context.Context, deviceID string, authCtx pbCQRS.AuthorizationContext, sequence uint64) error {
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
			ConnectionId: OpenapiConnectorConnectionId,
		},
	}

	_, err := s.raClient.PublishResource(ctx, &request)
	if err != nil {
		return fmt.Errorf("cannot process command publish resource: %v", err)
	}
	return nil
}

func (s *SubscribeManager) HandleDevicesRegistered(ctx context.Context, d subscriptionData, devices events.DevicesRegistered, header events.EventHeader) error {
	var errors []error
	userID, err := d.linkedAccount.OriginCloud.AccessToken.GetSubject()
	if err != nil {
		return fmt.Errorf("cannot get userID: %v", err)
	}
	for _, device := range devices {
		ctx := kitNetGrpc.CtxWithToken(ctx, d.linkedAccount.OriginCloud.AccessToken.String())
		_, err := s.asClient.AddDevice(ctx, &pbAS.AddDeviceRequest{
			DeviceId: device.ID,
			UserId:   userID,
		})
		if err != nil {
			errors = append(errors, err)
			continue
		}
		authCtx := pbCQRS.AuthorizationContext{
			UserId:   userID,
			DeviceId: device.ID,
		}

		err = s.publishCloudDeviceStatus(ctx, device.ID, authCtx, header.SequenceNumber)
		if err != nil {
			errors = append(errors, err)
			continue
		}

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
			LinkedAccountID: d.linkedAccount.ID,
			DeviceID:        device.ID,
			SigningSecret:   signingSecret,
		}
		err = s.cache.Add(correlationID, subscriptionData{
			linkedAccount: d.linkedAccount,
			subscription:  sub,
		}, cache.DefaultExpiration)
		if err != nil {
			return fmt.Errorf("cannot cache subscription for device subscriptions: %v", err)
		}
		sub.SubscriptionID, err = s.subscribeToDevice(ctx, d.linkedAccount, correlationID, signingSecret, device.ID)
		if err != nil {
			s.cache.Delete(correlationID)
			errors = append(errors, fmt.Errorf("cannot subscribe to device %v: %v", device.ID, err))
			continue
		}
		_, err = s.store.FindOrCreateSubscription(ctx, sub)
		if err != nil {
			cancelDevicesSubscription(ctx, d.linkedAccount, sub.SubscriptionID)
			errors = append(errors, fmt.Errorf("cannot store subscription to DB: %v", err))
			continue
		}
		loaded, err := s.resourceProjection.Register(ctx, device.ID)
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot register device %v to resource projection: %v", device.ID, err))
			continue
		}
		if !loaded {
			// we want to be only once registered in projection.
			s.resourceProjection.Unregister(device.ID)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

func (s *SubscribeManager) HandleDevicesUnregistered(ctx context.Context, subscriptionData subscriptionData, correlationID string, devices events.DevicesUnregistered) error {
	userID, err := subscriptionData.linkedAccount.OriginCloud.AccessToken.GetSubject()
	if err != nil {
		return fmt.Errorf("cannot get userID: %v", err)
	}
	var errors []error
	for _, device := range devices {
		err := cancelDeviceSubscription(ctx, subscriptionData.linkedAccount, device.ID, subscriptionData.subscription.SubscriptionID)
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot cancel subscription to device %v: %v", device.ID, err))
		}
		err = s.store.RemoveSubscriptions(ctx, store.SubscriptionQuery{SubscriptionID: subscriptionData.subscription.SubscriptionID})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot remove device %v subscription: %v", device.ID, err))
		}
		s.cache.Delete(correlationID)
		_, err = s.asClient.RemoveDevice(kitNetGrpc.CtxWithToken(ctx, subscriptionData.linkedAccount.OriginCloud.AccessToken.String()), &pbAS.RemoveDeviceRequest{
			DeviceId: device.ID,
			UserId:   userID,
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot remove device  %v from user: %v", device.ID, err))
		}

		err = s.resourceProjection.Unregister(device.ID)
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
func (s *SubscribeManager) HandleDevicesOnline(ctx context.Context, subscriptionData subscriptionData, header events.EventHeader, devices events.DevicesOnline) error {
	var errors []error
	for _, device := range devices {
		userID, err := subscriptionData.linkedAccount.OriginCloud.AccessToken.GetSubject()
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot get userID for set device(%v) online: %v", device.ID, err))
			continue
		}
		authCtx := pbCQRS.AuthorizationContext{
			UserId:   userID,
			DeviceId: device.ID,
		}

		err = s.updateCloudStatus(kitNetGrpc.CtxWithToken(ctx, subscriptionData.linkedAccount.OriginCloud.AccessToken.String()), device.ID, true, authCtx, header.SequenceNumber)

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
func (s *SubscribeManager) HandleDevicesOffline(ctx context.Context, subscriptionData subscriptionData, header events.EventHeader, devices events.DevicesOffline) error {
	var errors []error
	for _, device := range devices {
		userID, err := subscriptionData.linkedAccount.OriginCloud.AccessToken.GetSubject()
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot get userID for set device(%v) offline: %v", device.ID, err))
			continue
		}
		authCtx := pbCQRS.AuthorizationContext{
			UserId:   userID,
			DeviceId: device.ID,
		}

		err = s.updateCloudStatus(kitNetGrpc.CtxWithToken(ctx, subscriptionData.linkedAccount.OriginCloud.AccessToken.String()), device.ID, false, authCtx, header.SequenceNumber)

		if err != nil {
			errors = append(errors, fmt.Errorf("cannot set device %v to offline: %v", device.ID, err))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}

	return nil
}

func (s *SubscribeManager) HandleDevicesEvent(ctx context.Context, header events.EventHeader, body []byte, subscriptionData subscriptionData) error {
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
