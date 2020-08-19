package service

import (
	"context"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/kit/log"
)

type devicesSubsciptionHandler struct {
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

func (h *devicesSubsciptionHandler) HandleDeviceOnline(ctx context.Context, val *pb.Event_DeviceOnline) error {
	remove, err := h.emitEvent(ctx, events.EventType_DevicesOnline, h.subData.Data(), h.subData.IncrementSequenceNumber, makeDevicesRepresentation(val.GetDeviceIds()))
	if err != nil {
		log.Errorf("devicesSubsciptionHandler.HandleDeviceOnline: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

func (h *devicesSubsciptionHandler) HandleDeviceOffline(ctx context.Context, val *pb.Event_DeviceOffline) error {
	remove, err := h.emitEvent(ctx, events.EventType_DevicesOffline, h.subData.Data(), h.subData.IncrementSequenceNumber, makeDevicesRepresentation(val.GetDeviceIds()))
	if err != nil {
		log.Errorf("devicesSubsciptionHandler.HandleDeviceOffline: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

func (h *devicesSubsciptionHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	remove, err := h.emitEvent(ctx, events.EventType_DevicesRegistered, h.subData.Data(), h.subData.IncrementSequenceNumber, makeDevicesRepresentation(val.GetDeviceIds()))
	if err != nil {
		log.Errorf("devicesSubsciptionHandler.HandleDeviceRegistered: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

func (h *devicesSubsciptionHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	remove, err := h.emitEvent(ctx, events.EventType_DevicesUnregistered, h.subData.Data(), h.subData.IncrementSequenceNumber, makeDevicesRepresentation(val.GetDeviceIds()))
	if err != nil {
		log.Errorf("devicesSubsciptionHandler.HandleDeviceUnregistered: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

type devicesOnlineHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesOnlineHandler) HandleDeviceOnline(ctx context.Context, val *pb.Event_DeviceOnline) error {
	return h.h.HandleDeviceOnline(ctx, val)
}

type devicesOfflineHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesOfflineHandler) HandleDeviceOffline(ctx context.Context, val *pb.Event_DeviceOffline) error {
	return h.h.HandleDeviceOffline(ctx, val)
}

type devicesOnlineOfflineHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesOnlineOfflineHandler) HandleDeviceOnline(ctx context.Context, val *pb.Event_DeviceOnline) error {
	return h.h.HandleDeviceOnline(ctx, val)
}

func (h *devicesOnlineOfflineHandler) HandleDeviceOffline(ctx context.Context, val *pb.Event_DeviceOffline) error {
	return h.h.HandleDeviceOffline(ctx, val)
}

type devicesRegisteredHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesRegisteredHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

type devicesUnregisteredHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesUnregisteredHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

type devicesRegisteredUnregisteredHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesRegisteredUnregisteredHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

func (h *devicesRegisteredUnregisteredHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

type devicesRegisteredOnlineHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesRegisteredOnlineHandler) HandleDeviceOnline(ctx context.Context, val *pb.Event_DeviceOnline) error {
	return h.h.HandleDeviceOnline(ctx, val)
}

func (h *devicesRegisteredOnlineHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

type devicesRegisteredOfflineHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesRegisteredOfflineHandler) HandleDeviceOffline(ctx context.Context, val *pb.Event_DeviceOffline) error {
	return h.h.HandleDeviceOffline(ctx, val)
}

func (h *devicesRegisteredOfflineHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

type devicesUnregisteredOnlineHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesUnregisteredOnlineHandler) HandleDeviceOnline(ctx context.Context, val *pb.Event_DeviceOnline) error {
	return h.h.HandleDeviceOnline(ctx, val)
}

func (h *devicesUnregisteredOnlineHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

type devicesUnregisteredOfflineHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesUnregisteredOfflineHandler) HandleDeviceOffline(ctx context.Context, val *pb.Event_DeviceOffline) error {
	return h.h.HandleDeviceOffline(ctx, val)
}

func (h *devicesUnregisteredOfflineHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

type devicesRegisteredOnlineOfflineHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesRegisteredOnlineOfflineHandler) HandleDeviceOnline(ctx context.Context, val *pb.Event_DeviceOnline) error {
	return h.h.HandleDeviceOnline(ctx, val)
}

func (h *devicesRegisteredOnlineOfflineHandler) HandleDeviceOffline(ctx context.Context, val *pb.Event_DeviceOffline) error {
	return h.h.HandleDeviceOffline(ctx, val)
}

func (h *devicesRegisteredOnlineOfflineHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

type devicesUnregisteredOnlineOfflineHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesUnregisteredOnlineOfflineHandler) HandleDeviceOnline(ctx context.Context, val *pb.Event_DeviceOnline) error {
	return h.h.HandleDeviceOnline(ctx, val)
}

func (h *devicesUnregisteredOnlineOfflineHandler) HandleDeviceOffline(ctx context.Context, val *pb.Event_DeviceOffline) error {
	return h.h.HandleDeviceOffline(ctx, val)
}

func (h *devicesUnregisteredOnlineOfflineHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

type devicesRegisteredUnregisteredOfflineHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesRegisteredUnregisteredOfflineHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

func (h *devicesRegisteredUnregisteredOfflineHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

func (h *devicesRegisteredUnregisteredOfflineHandler) HandleDeviceOffline(ctx context.Context, val *pb.Event_DeviceOffline) error {
	return h.h.HandleDeviceOffline(ctx, val)
}

type devicesRegisteredUnregisteredOnlineHandler struct {
	h *devicesSubsciptionHandler
}

func (h *devicesRegisteredUnregisteredOnlineHandler) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return h.h.HandleDeviceRegistered(ctx, val)
}

func (h *devicesRegisteredUnregisteredOnlineHandler) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return h.h.HandleDeviceUnregistered(ctx, val)
}

func (h *devicesRegisteredUnregisteredOnlineHandler) HandleDeviceOnline(ctx context.Context, val *pb.Event_DeviceOnline) error {
	return h.h.HandleDeviceOnline(ctx, val)
}
