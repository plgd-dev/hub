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
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"go.opentelemetry.io/otel/trace"
)

const prefixDevicesPath = "/devices/"

func (s *SubscriptionManager) SubscribeToDevices(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	if _, loaded := s.store.LoadDevicesSubscription(linkedAccount.LinkedCloudID, linkedAccount.ID); loaded {
		return nil
	}
	signingSecret, err := generateRandomString(32)
	if err != nil {
		return fmt.Errorf("cannot generate signingSecret for start subscriptions: %w", err)
	}
	corID, err := uuid.NewRandom()
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
	data := SubscriptionData{
		linkedAccount: linkedAccount,
		linkedCloud:   linkedCloud,
		subscription:  sub,
	}
	_, loaded := s.cache.LoadOrStore(correlationID, cache.NewElement(data, time.Now().Add(CacheExpiration), nil))
	if loaded {
		return fmt.Errorf("cannot cache subscription for start subscriptions: subscription with %v already exists", correlationID)
	}
	sub.ID, err = s.subscribeToDevices(ctx, linkedAccount, linkedCloud, correlationID, signingSecret)
	if err != nil {
		s.cache.Delete(correlationID)
		return fmt.Errorf("cannot subscribe to devices for %v: %w", linkedAccount.ID, err)
	}
	_, _, err = s.store.LoadOrCreateSubscription(sub)
	if err != nil {
		var errors *multierror.Error
		errors = multierror.Append(errors, fmt.Errorf("cannot store subscription to DB: %w", err))
		if err2 := cancelDevicesSubscription(ctx, s.tracerProvider, linkedAccount, linkedCloud, sub.ID); err2 != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot cancel subscription %v: %w", sub.ID, err2))
		}
		return errors.ErrorOrNil()
	}
	return nil
}

func (s *SubscriptionManager) subscribeToDevices(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, correlationID, signingSecret string) (string, error) {
	resp, err := subscribe(ctx, s.tracerProvider, prefixDevicesPath+"subscriptions", correlationID, events.SubscriptionRequest{
		EventsURL: s.eventsURL,
		EventTypes: []events.EventType{
			events.EventType_DevicesRegistered, events.EventType_DevicesUnregistered,
			events.EventType_DevicesOnline, events.EventType_DevicesOffline,
		},
		SigningSecret: signingSecret,
	}, linkedAccount, linkedCloud)
	if err != nil {
		return "", err
	}
	return resp.SubscriptionID, nil
}

func cancelDevicesSubscription(ctx context.Context, tracerProvider trace.TracerProvider, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, subscriptionID string) error {
	err := cancelSubscription(ctx, tracerProvider, prefixDevicesPath+"subscriptions/"+subscriptionID, linkedAccount, linkedCloud)
	if err != nil {
		return fmt.Errorf("cannot cancel devices subscription for %v: %w", linkedAccount.ID, err)
	}
	return nil
}

func (s *SubscriptionManager) handleDevicesRegistered(ctx context.Context, d SubscriptionData, devices events.DevicesRegistered) error {
	var errors *multierror.Error
	for _, device := range devices {
		_, err := s.isClient.AddDevice(ctx, &pbIS.AddDeviceRequest{
			DeviceId: device.ID,
		})
		if err != nil {
			errors = multierror.Append(errors, err)
			continue
		}
		if d.linkedCloud.SupportedSubscriptionEvents.StaticDeviceEvents {
			s.triggerTask(Task{
				taskType:      TaskType_PullDevice,
				linkedAccount: d.linkedAccount,
				linkedCloud:   d.linkedCloud,
				deviceID:      device.ID,
			})
			continue
		}
		if d.linkedCloud.SupportedSubscriptionEvents.NeedPullDevice() {
			continue
		}
		s.triggerTask(Task{
			taskType:      TaskType_SubscribeToDevice,
			linkedAccount: d.linkedAccount,
			linkedCloud:   d.linkedCloud,
			deviceID:      device.ID,
		})
	}
	return errors.ErrorOrNil()
}

func (s *SubscriptionManager) handleDevicesUnregistered(ctx context.Context, subscriptionData SubscriptionData, correlationID string, devices events.DevicesUnregistered) error {
	userID := subscriptionData.linkedAccount.UserID
	var errors *multierror.Error
	for _, device := range devices {
		_, ok := s.store.PullOutDevice(subscriptionData.linkedAccount.LinkedCloudID, subscriptionData.linkedAccount.ID, device.ID)
		if !ok {
			log.Debugf("handleDevicesUnregistered: cannot remove device %v subscription: not found", device.ID)
		}
		s.cache.Delete(correlationID)
		resp, err := s.isClient.DeleteDevices(ctx, &pbIS.DeleteDevicesRequest{
			DeviceIds: []string{device.ID},
		})
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot remove device %v from user: %w", device.ID, err))
		}
		if err == nil && len(resp.DeviceIds) != 1 {
			errors = multierror.Append(errors, fmt.Errorf("cannot remove device %v from user", device.ID))
		}
		err = s.devicesSubscription.Delete(userID, device.ID)
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot unregister device %v from resource projection: %w", device.ID, err))
		}
	}
	return errors.ErrorOrNil()
}

// handleDevicesOnline sets device online to resource aggregate and register device to projection.
func (s *SubscriptionManager) handleDevicesOnline(ctx context.Context, d SubscriptionData, header events.EventHeader, devices events.DevicesOnline) error {
	var errors *multierror.Error
	for _, device := range devices {
		_, err := s.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
			DeviceId: device.ID,
			Update: &commands.UpdateDeviceMetadataRequest_Connection{
				Connection: &commands.Connection{
					Status:   commands.Connection_ONLINE,
					Protocol: commands.Connection_C2C,
				},
			},
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: d.linkedAccount.ID + "." + d.subscription.ID,
				Sequence:     header.SequenceNumber,
			},
		})
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot set device %v to online: %w", device.ID, err))
		}
	}

	return errors.ErrorOrNil()
}

// handleDevicesOffline sets device off to resource aggregate and unregister device to projection.
func (s *SubscriptionManager) handleDevicesOffline(ctx context.Context, d SubscriptionData, header events.EventHeader, devices events.DevicesOffline) error {
	var errors *multierror.Error
	for _, device := range devices {
		_, err := s.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
			DeviceId: device.ID,
			Update: &commands.UpdateDeviceMetadataRequest_Connection{
				Connection: &commands.Connection{
					Status: commands.Connection_OFFLINE,
				},
			},
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: d.linkedAccount.ID + "." + d.subscription.ID,
				Sequence:     header.SequenceNumber,
			},
		})
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot set device %v to offline: %w", device.ID, err))
		}
	}
	return errors.ErrorOrNil()
}

func decodeError(err error) error {
	return fmt.Errorf("cannot decode devices event: %w", err)
}

func (s *SubscriptionManager) handleDevicesEvent(ctx context.Context, header events.EventHeader, body []byte, d SubscriptionData) error {
	contentReader, err := header.GetContentDecoder()
	if err != nil {
		return fmt.Errorf("cannot handle device event: %w", err)
	}

	switch header.EventType {
	case events.EventType_DevicesRegistered:
		var devices events.DevicesRegistered
		err = contentReader(body, &devices)
		if err != nil {
			return decodeError(err)
		}
		return s.handleDevicesRegistered(ctx, d, devices)
	case events.EventType_DevicesUnregistered:
		var devices events.DevicesUnregistered
		err = contentReader(body, &devices)
		if err != nil {
			return decodeError(err)
		}
		return s.handleDevicesUnregistered(ctx, d, header.CorrelationID, devices)
	case events.EventType_DevicesOnline:
		var devices events.DevicesOnline
		err = contentReader(body, &devices)
		if err != nil {
			return decodeError(err)
		}
		return s.handleDevicesOnline(ctx, d, header, devices)
	case events.EventType_DevicesOffline:
		var devices events.DevicesOffline
		err = contentReader(body, &devices)
		if err != nil {
			return decodeError(err)
		}
		return s.handleDevicesOffline(ctx, d, header, devices)
	}

	return decodeError(fmt.Errorf("unsupported Event-Type %v", header.EventType))
}
