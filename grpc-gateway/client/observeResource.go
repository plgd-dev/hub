package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/events"
	"github.com/plgd-dev/sdk/v2/local/core"
)

func (c *Client) ObserveResource(
	ctx context.Context,
	deviceID string,
	href string,
	handler core.ObservationHandler,
	opts ...ObserveOption,
) (observationID string, _ error) {
	cfg := observeOptions{
		codec: GeneralMessageCodec{},
	}
	for _, o := range opts {
		cfg = o.applyOnObserve(cfg)
	}

	ID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	sub, err := c.NewResourceSubscription(ctx, commands.NewResourceID(deviceID, href), &observationHandler{
		codec: cfg.codec,
		obs:   handler,
		removeSubscription: func() {
			if _, err := c.stopObservingResource(ID.String()); err != nil {
				handler.Error(fmt.Errorf("failed to stop resource('%v') observation: %w", ID.String(), err))
			}
		},
	})
	if err != nil {
		return "", err
	}
	c.insertSubscription(ID.String(), sub)

	return ID.String(), err
}

func (c *Client) stopObservingResource(observationID string) (wait func(), err error) {
	s, err := c.popSubscription(observationID)
	if err != nil {
		return nil, err
	}
	return s.Cancel()
}

func (c *Client) StopObservingResource(ctx context.Context, observationID string) error {
	wait, err := c.stopObservingResource(observationID)
	if err != nil {
		return err
	}
	wait()
	return nil
}

type observationHandler struct {
	obs                core.ObservationHandler
	codec              Codec
	removeSubscription func()
}

func (o *observationHandler) HandleResourceContentChanged(ctx context.Context, ev *events.ResourceChanged) error {
	o.obs.Handle(ctx, func(v interface{}) error {
		return DecodeContentWithCodec(o.codec, ev.GetContent().GetContentType(), ev.GetContent().GetData(), v)
	})
	return nil
}

func (o *observationHandler) OnClose() {
	o.removeSubscription()
	o.obs.OnClose()
}

func (o *observationHandler) Error(err error) {
	o.removeSubscription()
	o.obs.Error(err)
}
