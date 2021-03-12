package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type deleteHandler struct {
	correlationID string
	res           chan *events.ResourceDeleted
}

func newDeleteHandler(correlationID string) *deleteHandler {
	return &deleteHandler{
		correlationID: correlationID,
		res:           make(chan *events.ResourceDeleted, 1),
	}
}

func (h *deleteHandler) Handle(ctx context.Context, iter eventbus.Iter) (err error) {
	for {
		ev, ok := iter.Next(ctx)
		if !ok {
			return iter.Err()
		}
		var s events.ResourceDeleted
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

func (h *deleteHandler) recv(ctx context.Context) (*events.ResourceDeleted, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case v := <-h.res:
		return v, nil
	}
}

// SyncDeleteResource sends delete resource command to resource aggregate and wait for resource deleted event from eventbus.
func (c *Client) SyncDeleteResource(ctx context.Context, req *commands.DeleteResourceRequest) (*events.ResourceDeleted, error) {
	h := newDeleteHandler(req.GetCorrelationId())
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), utils.GetTopics(req.GetResourceId().GetDeviceId()), h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to eventbus: %w", err)
	}
	defer obs.Close()

	_, err = c.DeleteResource(ctx, req)
	if err != nil {
		return nil, err
	}

	return h.recv(ctx)
}
