package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

type retrieveHandler struct {
	correlationID string
	res           chan *events.ResourceRetrieved
}

func newRetrieveHandler(correlationID string) *retrieveHandler {
	return &retrieveHandler{
		correlationID: correlationID,
		res:           make(chan *events.ResourceRetrieved, 1),
	}
}

func (h *retrieveHandler) Handle(ctx context.Context, iter eventbus.Iter) (err error) {
	for {
		ev, ok := iter.Next(ctx)
		if !ok {
			return iter.Err()
		}
		var s events.ResourceRetrieved
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

func (h *retrieveHandler) recv(ctx context.Context) (*events.ResourceRetrieved, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case v := <-h.res:
		return v, nil
	}
}

// SyncRetrieveResource sends retrieve resource command to resource aggregate and wait for resource retrieved event from eventbus.
func (c *Client) SyncRetrieveResource(ctx context.Context, owner string, req *commands.RetrieveResourceRequest) (*events.ResourceRetrieved, error) {
	h := newRetrieveHandler(req.GetCorrelationId())
	subject := utils.GetResourceEventSubject(owner, req.GetResourceId(), (&events.ResourceRetrieved{}).EventType())
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), subject, h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to eventbus: %w", err)
	}
	defer func() {
		if errC := obs.Close(); errC != nil {
			log.Errorf("retrieve resource: %w", errC)
		}
	}()

	_, err = c.RetrieveResource(ctx, req)
	if err != nil {
		return nil, err
	}

	return h.recv(ctx)
}
