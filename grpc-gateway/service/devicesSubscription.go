package service

import (
	"context"
	"fmt"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	cqrsRA "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	projectionRA "github.com/go-ocf/cloud/resource-aggregate/cqrs/projection"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/sdk/schema/cloud"
)

type devicesSubscription struct {
	*subscription
	devicesEvent *pb.SubscribeForEvents_DevicesEventFilter
}

func NewDevicesSubscription(id, userID string, send SendEventFunc, resourceProjection *projectionRA.Projection, devicesEvent *pb.SubscribeForEvents_DevicesEventFilter) *devicesSubscription {
	log.Debugf("subscription.NewDevicesSubscription %v", id)
	defer log.Debugf("subscription.NewDevicesSubscription %v done", id)
	return &devicesSubscription{
		subscription: NewSubscription(userID, id, send, resourceProjection),
		devicesEvent: devicesEvent,
	}
}

func (s *devicesSubscription) Init(ctx context.Context, currentDevices map[string]bool) error {
	for deviceID := range currentDevices {
		var notifyRegistered, notifyOnline, notifyOffline bool
		for _, f := range s.devicesEvent.GetFilterEvents() {
			switch f {
			case pb.SubscribeForEvents_DevicesEventFilter_REGISTERED:
				notifyRegistered = true
			case pb.SubscribeForEvents_DevicesEventFilter_ONLINE:
				notifyOnline = true
			case pb.SubscribeForEvents_DevicesEventFilter_OFFLINE:
				notifyOffline = true
			}
		}
		if notifyRegistered {
			err := s.NotifyOfRegisteredDevice(ctx, deviceID)
			if err != nil {
				return err
			}
		}
		_, err := s.RegisterToProjection(ctx, deviceID)
		if err != nil {
			log.Errorf("cannot register to resource projection for %v: %v", deviceID, err)
			continue
		}
		if notifyOnline {
			err = s.initNotifyOfOnlineDevice(ctx, deviceID)
			if err != nil {
				return err
			}
		}
		if notifyOffline {
			err = s.initNotifyOfOfflineDevice(ctx, deviceID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *devicesSubscription) Update(ctx context.Context, addedDevices, removedDevices map[string]bool) error {
	for deviceID := range removedDevices {
		err := s.UnregisterFromProjection(ctx, deviceID)
		if err != nil {
			log.Errorf("cannot unregister resource from projection for %v: %v", deviceID, err)
		}
		for _, f := range s.devicesEvent.GetFilterEvents() {
			switch f {
			case pb.SubscribeForEvents_DevicesEventFilter_UNREGISTERED:
				err = s.NotifyOfUnregisteredDevice(ctx, deviceID)
				if err != nil {
					return fmt.Errorf("cannot send device unregistered: %w", err)
				}
			}
		}
	}
	return s.Init(ctx, addedDevices)
}

func (s *devicesSubscription) NotifyOfRegisteredDevice(ctx context.Context, deviceID string) error {
	return s.Send(ctx, pb.Event{
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceRegistered_{
			DeviceRegistered: &pb.Event_DeviceRegistered{
				DeviceId: deviceID,
			},
		},
	})
}

func (s *devicesSubscription) NotifyOfUnregisteredDevice(ctx context.Context, deviceID string) error {
	return s.Send(ctx, pb.Event{
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceUnregistered_{
			DeviceUnregistered: &pb.Event_DeviceUnregistered{
				DeviceId: deviceID,
			},
		},
	})
}

func (s *devicesSubscription) NotifyOfOnlineDevice(ctx context.Context, deviceID string, version uint64) error {
	if s.FilterByVersion(deviceID, cloud.StatusHref, "devStatus", version) {
		return nil
	}
	var found bool
	for _, f := range s.devicesEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DevicesEventFilter_ONLINE {
			found = true
		}
	}
	if !found {
		return nil
	}
	return s.Send(ctx, pb.Event{
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceOnline_{
			DeviceOnline: &pb.Event_DeviceOnline{
				DeviceId: deviceID,
			},
		},
	})
}

func (s *devicesSubscription) NotifyOfOfflineDevice(ctx context.Context, deviceID string, version uint64) error {
	if s.FilterByVersion(deviceID, cloud.StatusHref, "devStatus", version) {
		return nil
	}
	var found bool
	for _, f := range s.devicesEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DevicesEventFilter_OFFLINE {
			found = true
		}
	}
	if !found {
		return nil
	}
	return s.Send(ctx, pb.Event{
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceOffline_{
			DeviceOffline: &pb.Event_DeviceOffline{
				DeviceId: deviceID,
			},
		},
	})
}

func (s *devicesSubscription) initNotifyOfOnlineDevice(ctx context.Context, deviceID string) error {
	cloudResourceID := cqrsRA.MakeResourceId(deviceID, cloud.StatusHref)
	models := s.resourceProjection.Models(deviceID, cloudResourceID)
	if len(models) == 0 {
		return nil
	}
	res := models[0].(*resourceCtx).Clone()
	online, err := isDeviceOnline(res.content.GetContent())
	if err != nil {
		return fmt.Errorf("cannot determine device cloud status: %w", err)

	}
	if !online {
		return nil
	}
	err = s.NotifyOfOnlineDevice(ctx, deviceID, res.onResourceChangedVersion)
	if err != nil {
		return fmt.Errorf("cannot send device online: %w", err)
	}
	return nil
}

func (s *devicesSubscription) initNotifyOfOfflineDevice(ctx context.Context, deviceID string) error {
	cloudResourceID := cqrsRA.MakeResourceId(deviceID, cloud.StatusHref)
	models := s.resourceProjection.Models(deviceID, cloudResourceID)
	if len(models) == 0 {
		return nil
	}
	res := models[0].(*resourceCtx).Clone()
	online, err := isDeviceOnline(res.content.GetContent())
	if err != nil {
		return fmt.Errorf("cannot determine device cloud status: %w", err)

	}
	if online {
		return nil
	}
	err = s.NotifyOfOfflineDevice(ctx, deviceID, res.onResourceChangedVersion)
	if err != nil {
		return fmt.Errorf("cannot send device offline: %w", err)
	}
	return nil
}
