package client

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
)

type DevicesObservationEvent_type uint8

const DevicesObservationEvent_ONLINE DevicesObservationEvent_type = 0
const DevicesObservationEvent_OFFLINE DevicesObservationEvent_type = 1
const DevicesObservationEvent_REGISTERED DevicesObservationEvent_type = 2
const DevicesObservationEvent_UNREGISTERED DevicesObservationEvent_type = 3

type DevicesObservationEvent struct {
	DeviceIDs []string
	Event     DevicesObservationEvent_type
}

type DevicesObservationHandler = interface {
	Handle(ctx context.Context, event DevicesObservationEvent) error
	OnClose()
	Error(err error)
}

type devicesObservation struct {
	h                  DevicesObservationHandler
	removeSubscription func()
}

func (o *devicesObservation) HandleDeviceOnline(ctx context.Context, val *pb.Event_DeviceOnline) error {
	return o.h.Handle(ctx, DevicesObservationEvent{
		DeviceIDs: val.GetDeviceIds(),
		Event:     DevicesObservationEvent_ONLINE,
	})
}

func (o *devicesObservation) HandleDeviceOffline(ctx context.Context, val *pb.Event_DeviceOffline) error {
	return o.h.Handle(ctx, DevicesObservationEvent{
		DeviceIDs: val.GetDeviceIds(),
		Event:     DevicesObservationEvent_OFFLINE,
	})
}

func (o *devicesObservation) HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error {
	return o.h.Handle(ctx, DevicesObservationEvent{
		DeviceIDs: val.GetDeviceIds(),
		Event:     DevicesObservationEvent_REGISTERED,
	})
}

func (o *devicesObservation) HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error {
	return o.h.Handle(ctx, DevicesObservationEvent{
		DeviceIDs: val.GetDeviceIds(),
		Event:     DevicesObservationEvent_UNREGISTERED,
	})
}

func (o *devicesObservation) OnClose() {
	o.removeSubscription()
	o.h.OnClose()
}
func (o *devicesObservation) Error(err error) {
	o.removeSubscription()
	o.h.Error(err)
}

func (c *Client) ObserveDevices(ctx context.Context, handler DevicesObservationHandler) (string, error) {
	ID, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	sub, err := c.NewDevicesSubscription(ctx, &devicesObservation{
		h: handler,
		removeSubscription: func() {
			c.stopObservingDevices(ID.String())
		},
	})
	if err != nil {
		return "", err
	}
	c.insertSubscription(ID.String(), sub)

	return ID.String(), err
}

func (c *Client) stopObservingDevices(observationID string) (wait func(), err error) {
	s, err := c.popSubscription(observationID)
	if err != nil {
		return nil, err
	}
	return s.Cancel()
}

func (c *Client) StopObservingDevices(ctx context.Context, observationID string) error {
	wait, err := c.stopObservingDevices(observationID)
	if err != nil {
		return err
	}
	wait()
	return nil
}
