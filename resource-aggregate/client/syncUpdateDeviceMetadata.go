package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type updateDeviceMetadataHandler struct {
	correlationID string
	res           chan *events.DeviceMetadataUpdated
}

func newUpdateDeviceMetadataHandler(correlationID string) *updateDeviceMetadataHandler {
	return &updateDeviceMetadataHandler{
		correlationID: correlationID,
		res:           make(chan *events.DeviceMetadataUpdated, 1),
	}
}

func (h *updateDeviceMetadataHandler) Handle(ctx context.Context, iter eventbus.Iter) (err error) {
	for {
		ev, ok := iter.Next(ctx)
		if !ok {
			return iter.Err()
		}
		var s events.DeviceMetadataUpdated
		if ev.EventType() == s.EventType() {
			if err := ev.Unmarshal(&s); err != nil {
				return err
			}
			if s.GetAuditContext().GetCorrelationId() == h.correlationID {
				select {
				case h.res <- &s:
				default:
				}
				return nil
			}
		}
	}
}

func (h *updateDeviceMetadataHandler) recv(ctx context.Context) (*events.DeviceMetadataUpdated, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case v := <-h.res:
		return v, nil
	}
}

// SyncUpdateDeviceMetadata sends update device metadata command to resource aggregate and wait for metadata updated event from eventbus.
func (c *Client) SyncUpdateDeviceMetadata(ctx context.Context, req *commands.UpdateDeviceMetadataRequest) (*events.DeviceMetadataUpdated, error) {
	h := newUpdateDeviceMetadataHandler(req.GetCorrelationId())
	subject := utils.GetDeviceMetadataEventSubject(req.GetDeviceId(), (&events.DeviceMetadataUpdated{}).EventType())
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), subject, h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to eventbus: %w", err)
	}
	defer obs.Close()

	_, err = c.UpdateDeviceMetadata(ctx, req)
	if err != nil {
		return nil, err
	}

	return h.recv(ctx)
}
