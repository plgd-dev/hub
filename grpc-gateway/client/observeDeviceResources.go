package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

type DeviceResourcesObservationHandler = interface {
	HandleResourcePublished(ctx context.Context, val *events.ResourceLinksPublished) error
	HandleResourceUnpublished(ctx context.Context, val *events.ResourceLinksUnpublished) error
	OnClose()
	Error(err error)
}

type deviceResourcesObservation struct {
	h                  DeviceResourcesObservationHandler
	removeSubscription func()
}

func (o *deviceResourcesObservation) HandleResourcePublished(ctx context.Context, val *events.ResourceLinksPublished) error {
	return o.h.HandleResourcePublished(ctx, val)
}

func (o *deviceResourcesObservation) HandleResourceUnpublished(ctx context.Context, val *events.ResourceLinksUnpublished) error {
	return o.h.HandleResourceUnpublished(ctx, val)
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
	ID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	sub, err := c.NewDeviceSubscription(ctx, deviceID, &deviceResourcesObservation{
		h: handler,
		removeSubscription: func() {
			if _, err := c.stopObservingDeviceResources(ID.String()); err != nil {
				handler.Error(fmt.Errorf("failed to stop device('%v') resources observation: %w", ID.String(), err))
			}
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
