package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/kit/log"
)

type devicesSubscription struct {
	*subscription
	devicesEvent *pb.SubscribeForEvents_DevicesEventFilter
}

func NewDevicesSubscription(id, userID, token string, send SendEventFunc, resourceProjection *Projection, devicesEvent *pb.SubscribeForEvents_DevicesEventFilter) *devicesSubscription {
	log.Debugf("subscription.NewDevicesSubscription %v", id)
	defer log.Debugf("subscription.NewDevicesSubscription %v done", id)
	return &devicesSubscription{
		subscription: NewSubscription(userID, id, token, send, resourceProjection),
		devicesEvent: devicesEvent,
	}
}

func (s *devicesSubscription) update(ctx context.Context, currentDevices map[string]bool, init bool) error {
	registeredDevices := make([]string, 0, 32)
	onlineDevices := make([]string, 0, 32)
	offlineDevices := make([]string, 0, 32)
	for deviceID := range currentDevices {
		registeredDevices = append(registeredDevices, deviceID)
		_, err := s.RegisterToProjection(ctx, deviceID)
		if err != nil {
			log.Errorf("cannot register to resource projection for %v: %v", deviceID, err)
			continue
		}
		onlineDevices = append(onlineDevices, deviceID)
		offlineDevices = append(offlineDevices, deviceID)
	}

	if init || len(registeredDevices) > 0 {
		err := s.NotifyOfRegisteredDevice(ctx, registeredDevices)
		if err != nil {
			return err
		}
	}
	if init {
		err := s.NotifyOfUnregisteredDevice(ctx, []string{})
		if err != nil {
			return err
		}
	}
	err := s.initNotifyOfOnlineDevice(ctx, onlineDevices)
	if err != nil {
		return err
	}
	err = s.initNotifyOfOfflineDevice(ctx, offlineDevices)
	if err != nil {
		return err
	}
	return nil
}

func (s *devicesSubscription) Init(ctx context.Context, currentDevices map[string]bool) error {
	return s.update(ctx, currentDevices, true)
}

func (s *devicesSubscription) Update(ctx context.Context, addedDevices, removedDevices map[string]bool) error {
	toSend := make([]string, 0, 32)
	for deviceID := range removedDevices {
		devID := deviceID
		toSend = append(toSend, devID)
		err := s.UnregisterFromProjection(ctx, deviceID)
		if err != nil {
			log.Errorf("cannot unregister resource from projection for %v: %v", deviceID, err)
		}
	}
	if len(toSend) > 0 {
		err := s.NotifyOfUnregisteredDevice(ctx, toSend)
		if err != nil {
			return fmt.Errorf("cannot send device unregistered: %w", err)
		}
	}
	return s.update(ctx, addedDevices, false)
}

func (s *devicesSubscription) NotifyOfRegisteredDevice(ctx context.Context, deviceIDs []string) error {
	var found bool
	for _, f := range s.devicesEvent.GetFilterEvents() {
		switch f {
		case pb.SubscribeForEvents_DevicesEventFilter_REGISTERED:
			found = true
		}
	}
	if !found {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceRegistered_{
			DeviceRegistered: &pb.Event_DeviceRegistered{
				DeviceIds: deviceIDs,
			},
		},
	})
}

func (s *devicesSubscription) NotifyOfUnregisteredDevice(ctx context.Context, deviceIDs []string) error {
	var found bool
	for _, f := range s.devicesEvent.GetFilterEvents() {
		switch f {
		case pb.SubscribeForEvents_DevicesEventFilter_UNREGISTERED:
			found = true
		}
	}
	if !found {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceUnregistered_{
			DeviceUnregistered: &pb.Event_DeviceUnregistered{
				DeviceIds: deviceIDs,
			},
		},
	})
}

type DeviceIDVersion struct {
	deviceID string
	version  uint64
}

func (s *devicesSubscription) NotifyOfOnlineDevice(ctx context.Context, devs []DeviceIDVersion) error {
	var found bool
	for _, f := range s.devicesEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DevicesEventFilter_ONLINE {
			found = true
		}
	}
	if !found {
		return nil
	}
	toSend := make([]string, 0, 32)
	for _, d := range devs {
		if s.FilterByVersion(d.deviceID, commands.StatusHref, "devStatus", d.version) {
			continue
		}
		toSend = append(toSend, d.deviceID)
	}
	if len(toSend) == 0 && len(devs) > 0 {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceOnline_{
			DeviceOnline: &pb.Event_DeviceOnline{
				DeviceIds: toSend,
			},
		},
	})
}

func (s *devicesSubscription) NotifyOfOfflineDevice(ctx context.Context, devs []DeviceIDVersion) error {
	var found bool
	for _, f := range s.devicesEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DevicesEventFilter_OFFLINE {
			found = true
		}
	}
	if !found {
		return nil
	}
	toSend := make([]string, 0, 32)
	for _, d := range devs {
		if s.FilterByVersion(d.deviceID, commands.StatusHref, "devStatus", d.version) {
			continue
		}
		toSend = append(toSend, d.deviceID)
	}
	if len(toSend) == 0 && len(devs) > 0 {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceOffline_{
			DeviceOffline: &pb.Event_DeviceOffline{
				DeviceIds: toSend,
			},
		},
	})
}

func (s *devicesSubscription) initNotifyOfOnlineDevice(ctx context.Context, deviceIDs []string) error {
	toSend := make([]DeviceIDVersion, 0, 32)
	for _, deviceID := range deviceIDs {
		statusResourceID := commands.NewResourceID(deviceID, commands.StatusHref)
		models := s.resourceProjection.Models(statusResourceID)
		if len(models) == 0 {
			continue
		}
		res := models[0].(*resourceProjection).Clone()
		online, err := isDeviceOnline(res.content.GetContent())
		if err != nil {
			log.Errorf("cannot determine device cloud status: %v", err)
			continue
		}
		if !online {
			continue
		}
		dID := deviceID
		toSend = append(toSend, DeviceIDVersion{
			deviceID: dID,
			version:  res.onResourceChangedVersion,
		})
	}
	err := s.NotifyOfOnlineDevice(ctx, toSend)
	if err != nil {
		return fmt.Errorf("cannot send device online: %w", err)
	}
	return nil
}

func (s *devicesSubscription) initNotifyOfOfflineDevice(ctx context.Context, deviceIDs []string) error {
	toSend := make([]DeviceIDVersion, 0, 32)
	for _, deviceID := range deviceIDs {
		statusResourceID := commands.NewResourceID(deviceID, commands.StatusHref)
		models := s.resourceProjection.Models(statusResourceID)
		if len(models) == 0 {
			continue
		}
		res := models[0].(*resourceProjection).Clone()
		online, err := isDeviceOnline(res.content.GetContent())
		if err != nil {
			log.Errorf("cannot determine device cloud status: %v", err)
			continue
		}
		if online {
			continue
		}
		dID := deviceID
		toSend = append(toSend, DeviceIDVersion{
			deviceID: dID,
			version:  res.onResourceChangedVersion,
		})
	}
	err := s.NotifyOfOfflineDevice(ctx, toSend)
	if err != nil {
		return fmt.Errorf("cannot send device offline: %w", err)
	}
	return nil
}
