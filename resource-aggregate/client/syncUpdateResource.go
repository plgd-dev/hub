package client

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

type updateHandler struct {
	correlationID string
	res           chan *events.ResourceUpdated
}

func newUpdateHandler(correlationID string) *updateHandler {
	return &updateHandler{
		correlationID: correlationID,
		res:           make(chan *events.ResourceUpdated, 1),
	}
}

func (h *updateHandler) Handle(ctx context.Context, iter eventbus.Iter) (err error) {
	for {
		ev, ok := iter.Next(ctx)
		if !ok {
			return iter.Err()
		}
		var s events.ResourceUpdated
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

func (h *updateHandler) recv(ctx context.Context) (*events.ResourceUpdated, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case v := <-h.res:
		return v, nil
	}
}

// SyncUpdateResource sends update resource command to resource aggregate and wait for resource updated event from eventbus.
func (c *Client) SyncUpdateResource(ctx context.Context, owner string, req *commands.UpdateResourceRequest) (*events.ResourceUpdated, error) {
	h := newUpdateHandler(req.GetCorrelationId())
	subject := c.subscriber.GetResourceEventSubjects(owner, req.GetResourceId(), (&events.ResourceUpdated{}).EventType())
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), subject, h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to eventbus: %w", err)
	}
	defer func() {
		if errC := obs.Close(); errC != nil {
			log.Errorf("update resource: %w", errC)
		}
	}()

	resp, err := c.UpdateResource(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetValidUntil() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, time.Unix(0, resp.GetValidUntil()))
		defer cancel()
	}

	return h.recv(ctx)
}
