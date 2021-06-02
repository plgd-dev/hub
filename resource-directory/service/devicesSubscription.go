package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/kit/strings"
)

type devicesSubscription struct {
	*subscription
	devicesEvent      *pb.SubscribeToEvents_DevicesEventFilter
	isInitialized     sync.Map
	filteredDeviceIDs strings.Set
}

func NewDevicesSubscription(id, userID, token string, send SendEventFunc, resourceProjection *Projection, devicesEvent *pb.SubscribeToEvents_DevicesEventFilter) *devicesSubscription {
	log.Debugf("subscription.NewDevicesSubscription %v", id)
	defer log.Debugf("subscription.NewDevicesSubscription %v done", id)
	return &devicesSubscription{
		subscription:      NewSubscription(userID, id, token, send, resourceProjection),
		devicesEvent:      devicesEvent,
		filteredDeviceIDs: strings.MakeSet(devicesEvent.GetDeviceIdsFilter()...),
	}
}

func (s *devicesSubscription) update(ctx context.Context, currentDevices map[string]bool, init bool) error {
	registeredDevices := make([]string, 0, 32)
	filteredDevices := make([]string, 0, 32)
	for deviceID := range currentDevices {
		registeredDevices = append(registeredDevices, deviceID)
		_, err := s.RegisterToProjection(ctx, deviceID)
		if err != nil {
			log.Errorf("cannot register to resource projection for %v: %v", deviceID, err)
			continue
		}
		if isFilteredDevice(s.filteredDeviceIDs, deviceID) {
			filteredDevices = append(filteredDevices, deviceID)
		}

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
	err := s.initNotifyOfDevicesMetadata(ctx, filteredDevices)
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
	for _, f := range s.devicesEvent.GetEventsFilter() {
		switch f {
		case pb.SubscribeToEvents_DevicesEventFilter_REGISTERED:
			found = true
		}
	}
	for _, d := range deviceIDs {
		s.isInitialized.Store(d, true)
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
	for _, f := range s.devicesEvent.GetEventsFilter() {
		switch f {
		case pb.SubscribeToEvents_DevicesEventFilter_UNREGISTERED:
			found = true
		}
	}
	for _, d := range deviceIDs {
		s.isInitialized.Delete(d)
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

func (s *devicesSubscription) initNotifyOfDevicesMetadata(ctx context.Context, deviceIDs []string) error {
	var errors []error
	for _, deviceID := range deviceIDs {
		statusResourceID := commands.NewResourceID(deviceID, commands.StatusHref)
		models := s.resourceProjection.Models(statusResourceID)
		if len(models) == 0 {
			continue
		}
		res := models[0].(*deviceMetadataProjection)
		err := res.InitialNotifyOfDeviceMetadata(ctx, s)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return nil
}

func (s *devicesSubscription) NotifyOfUpdatePendingDeviceMetadata(ctx context.Context, event *events.DeviceMetadataUpdatePending) error {
	var found bool
	if !isFilteredDevice(s.filteredDeviceIDs, event.GetDeviceId()) {
		return nil
	}
	for _, f := range s.devicesEvent.GetEventsFilter() {
		if f == pb.SubscribeToEvents_DevicesEventFilter_DEVICE_METADATA_UPDATE_PENDING {
			found = true
		}
	}
	if !found {
		return nil
	}
	if _, ok := s.isInitialized.Load(event.GetDeviceId()); !ok {
		return nil
	}
	if s.FilterByVersionAndHash(event.GetDeviceId(), commands.StatusHref, "res", event.Version(), 0) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceMetadataUpdatePending{
			DeviceMetadataUpdatePending: event,
		},
	})
}

func isFilteredDevice(filteredDeviceIDs strings.Set, deviceID string) bool {
	if len(filteredDeviceIDs) == 0 {
		return true
	}
	return filteredDeviceIDs.HasOneOf(deviceID)
}

func (s *devicesSubscription) NotifyOfUpdatedDeviceMetadata(ctx context.Context, event *events.DeviceMetadataUpdated) error {
	var found bool
	if !isFilteredDevice(s.filteredDeviceIDs, event.GetDeviceId()) {
		return nil
	}
	for _, f := range s.devicesEvent.GetEventsFilter() {
		if f == pb.SubscribeToEvents_DevicesEventFilter_DEVICE_METADATA_UPDATED {
			found = true
		}
	}
	if !found {
		return nil
	}
	if _, ok := s.isInitialized.Load(event.GetDeviceId()); !ok {
		return nil
	}
	if s.FilterByVersionAndHash(event.GetDeviceId(), commands.StatusHref, "res", event.Version(), 0) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: event,
		},
	})
}
