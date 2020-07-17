package client

import (
	"context"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/gofrs/uuid"
)

type DeviceResourcesObservationEvent_type uint8

const DeviceResourcesObservationEvent_ADDED DeviceResourcesObservationEvent_type = 0
const DeviceResourcesObservationEvent_REMOVED DeviceResourcesObservationEvent_type = 1

type DeviceResourcesObservationEvent struct {
	Links []*pb.ResourceLink
	Event DeviceResourcesObservationEvent_type
}

type DeviceResourcesObservationHandler = interface {
	Handle(ctx context.Context, event DeviceResourcesObservationEvent) error
	OnClose()
	Error(err error)
}

type deviceResourcesObservation struct {
	h                  DeviceResourcesObservationHandler
	removeSubscription func()
}

func (o *deviceResourcesObservation) HandleResourcePublished(ctx context.Context, val *pb.Event_ResourcePublished) error {
	return o.h.Handle(ctx, DeviceResourcesObservationEvent{
		Links: val.GetLinks(),
		Event: DeviceResourcesObservationEvent_ADDED,
	})
}

func (o *deviceResourcesObservation) HandleResourceUnpublished(ctx context.Context, val *pb.Event_ResourceUnpublished) error {
	return o.h.Handle(ctx, DeviceResourcesObservationEvent{
		Links: val.GetLinks(),
		Event: DeviceResourcesObservationEvent_REMOVED,
	})
}

func (o *deviceResourcesObservation) OnClose() {
	o.removeSubscription()
	o.h.OnClose()
}
func (o *deviceResourcesObservation) Error(err error) {
	o.removeSubscription()
	o.h.Error(err)
}

func (c *Client) ObserveDeviceResources(ctx context.Context, deviceID string, handler DeviceResourcesObservationHandler) (string, error) {
	ID, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	sub, err := c.NewDeviceSubscription(ctx, deviceID, &deviceResourcesObservation{
		h: handler,
		removeSubscription: func() {
			c.stopObservingDeviceResources(ID.String())
		},
	})
	if err != nil {
		return "", err
	}
	c.insertSubscription(ID.String(), sub)

	return ID.String(), err
}

func (c *Client) stopObservingDeviceResources(observationID string) (wait func(), err error) {
	s, err := c.popSubscription(observationID)
	if err != nil {
		return nil, err
	}
	return s.Cancel()
}

func (c *Client) StopObservingDeviceResources(ctx context.Context, observationID string) error {
	wait, err := c.stopObservingDeviceResources(observationID)
	if err != nil {
		return err
	}
	wait()
	return nil
}
