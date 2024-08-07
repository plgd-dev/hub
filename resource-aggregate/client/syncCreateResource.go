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

type createHandler struct {
	correlationID string
	res           chan *events.ResourceCreated
}

func newCreateHandler(correlationID string) *createHandler {
	return &createHandler{
		correlationID: correlationID,
		res:           make(chan *events.ResourceCreated, 1),
	}
}

func (h *createHandler) Handle(ctx context.Context, iter eventbus.Iter) (err error) {
	for {
		ev, ok := iter.Next(ctx)
		if !ok {
			return iter.Err()
		}
		var s events.ResourceCreated
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

func (h *createHandler) recv(ctx context.Context) (*events.ResourceCreated, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case v := <-h.res:
		return v, nil
	}
}

// SyncCreateResource sends create resource command to resource aggregate and wait for resource created event from eventbus.
func (c *Client) SyncCreateResource(ctx context.Context, owner string, req *commands.CreateResourceRequest) (*events.ResourceCreated, error) {
	h := newCreateHandler(req.GetCorrelationId())
	subject := c.subscriber.GetResourceEventSubjects(owner, req.GetResourceId(), (&events.ResourceCreated{}).EventType())
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), subject, h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to eventbus: %w", err)
	}
	defer func() {
		if errC := obs.Close(); errC != nil {
			log.Errorf("create resource: %w", errC)
		}
	}()

	resp, err := c.CreateResource(ctx, req)
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
