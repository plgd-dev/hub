package client

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	codecOcf "github.com/plgd-dev/kit/codec/ocf"
	kitNetCoap "github.com/plgd-dev/kit/net/coap"
	"github.com/plgd-dev/sdk/local/core"
)

func (c *Client) ObserveResource(
	ctx context.Context,
	deviceID string,
	href string,
	handler core.ObservationHandler,
	opts ...ObserveOption,
) (observationID string, _ error) {
	cfg := observeOptions{
		codec: codecOcf.VNDOCFCBORCodec{},
	}
	for _, o := range opts {
		cfg = o.applyOnObserve(cfg)
	}

	ID, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	sub, err := c.NewResourceSubscription(ctx, pb.ResourceId{
		DeviceId: deviceID,
		Href:     href,
	}, &observationHandler{
		codec: cfg.codec,
		obs:   handler,
		removeSubscription: func() {
			c.stopObservingResource(ID.String())
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
	codec              kitNetCoap.Codec
	removeSubscription func()
}

func (o *observationHandler) HandleResourceContentChanged(ctx context.Context, ev *pb.Event_ResourceChanged) error {
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
