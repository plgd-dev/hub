package service

import (
	"context"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/kit/log"

	raEvents "github.com/plgd-dev/cloud/resource-aggregate/events"
)

type devicesSubscriptionHandler struct {
	subData   *SubscriptionData
	emitEvent emitEventFunc
}

func makeDevicesRepresentation(deviceIDs []string) []map[string]string {
	devices := make([]map[string]string, 0, 32)
	for _, ID := range deviceIDs {
		devices = append(devices, map[string]string{"di": ID})
	}
	return devices
}

func (h *devicesSubscriptionHandler) HandleDeviceMetadataUpdated(ctx context.Context, val *raEvents.DeviceMetadataUpdated) error {
	if val.GetStatus() == nil {
		return nil
	}
	status := events.EventType_DevicesOffline
	if val.GetStatus().IsOnline() {
		status = events.EventType_DevicesOnline
	}

	remove, err := h.emitEvent(ctx, status, h.subData.Data(), h.subData.IncrementSequenceNumber, makeDevicesRepresentation([]string{val.GetDeviceId()}))
	if err != nil {
		log.Errorf("devicesSubscriptionHandler.HandleDeviceMetadataUpdated: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

func (h *devicesSubscriptionHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	remove, err := h.emitEvent(ctx, events.EventType_DevicesRegistered, h.subData.Data(), h.subData.IncrementSequenceNumber, makeDevicesRepresentation(val.GetDeviceIds()))
	if err != nil {
		log.Errorf("devicesSubscriptionHandler.HandleDeviceRegistered: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

func (h *devicesSubscriptionHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	remove, err := h.emitEvent(ctx, events.EventType_DevicesUnregistered, h.subData.Data(), h.subData.IncrementSequenceNumber, makeDevicesRepresentation(val.GetDeviceIds()))
	if err != nil {
		log.Errorf("devicesSubscriptionHandler.HandleDeviceUnregistered: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

type devicesOnlineHandler struct {
	h *devicesSubscriptionHandler
}

func isOnline(val *raEvents.DeviceMetadataUpdated) bool {
	if val.GetStatus() == nil {
		return false
	}
	return val.GetStatus().IsOnline()
}

func (h *devicesOnlineHandler) HandleDeviceMetadataUpdated(ctx context.Context, val *raEvents.DeviceMetadataUpdated) error {
	if !isOnline(val) {
		return nil
	}
	return h.h.HandleDeviceMetadataUpdated(ctx, val)
}

type devicesOfflineHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesOfflineHandler) HandleDeviceMetadataUpdated(ctx context.Context, val *raEvents.DeviceMetadataUpdated) error {
	if isOnline(val) {
		return nil
	}
	return h.h.HandleDeviceMetadataUpdated(ctx, val)
}

type devicesOnlineOfflineHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesOnlineOfflineHandler) HandleDeviceMetadataUpdated(ctx context.Context, val *raEvents.DeviceMetadataUpdated) error {
	return h.h.HandleDeviceMetadataUpdated(ctx, val)
}

type devicesRegisteredHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesRegisteredHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

type devicesUnregisteredHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesUnregisteredHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

type devicesRegisteredUnregisteredHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesRegisteredUnregisteredHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

func (h *devicesRegisteredUnregisteredHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

type devicesRegisteredOnlineHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesRegisteredOnlineHandler) HandleDeviceMetadataUpdated(ctx context.Context, val *raEvents.DeviceMetadataUpdated) error {
	if !isOnline(val) {
		return nil
	}
	return h.h.HandleDeviceMetadataUpdated(ctx, val)
}

func (h *devicesRegisteredOnlineHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

type devicesRegisteredOfflineHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesRegisteredOfflineHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

type devicesUnregisteredOnlineHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesUnregisteredOnlineHandler) HandleDeviceOnline(ctx context.Context, val *raEvents.DeviceMetadataUpdated) error {
	if !isOnline(val) {
		return nil
	}
	return h.h.HandleDeviceMetadataUpdated(ctx, val)
}

func (h *devicesUnregisteredOnlineHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

type devicesUnregisteredOfflineHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesUnregisteredOfflineHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

type devicesRegisteredOnlineOfflineHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesRegisteredOnlineOfflineHandler) HandleDeviceMetadataUpdated(ctx context.Context, val *raEvents.DeviceMetadataUpdated) error {
	return h.h.HandleDeviceMetadataUpdated(ctx, val)
}

func (h *devicesRegisteredOnlineOfflineHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

type devicesUnregisteredOnlineOfflineHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesUnregisteredOnlineOfflineHandler) HandleDeviceMetadataUpdated(ctx context.Context, val *raEvents.DeviceMetadataUpdated) error {
	return h.h.HandleDeviceMetadataUpdated(ctx, val)
}

func (h *devicesUnregisteredOnlineOfflineHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

type devicesRegisteredUnregisteredOfflineHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesRegisteredUnregisteredOfflineHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

func (h *devicesRegisteredUnregisteredOfflineHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

type devicesRegisteredUnregisteredOnlineHandler struct {
	h *devicesSubscriptionHandler
}

func (h *devicesRegisteredUnregisteredOnlineHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

func (h *devicesRegisteredUnregisteredOnlineHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

func (h *devicesRegisteredUnregisteredOnlineHandler) HandleDeviceMetadataUpdated(ctx context.Context, val *raEvents.DeviceMetadataUpdated) error {
	if !isOnline(val) {
		return nil
	}
	return h.h.HandleDeviceMetadataUpdated(ctx, val)
}
