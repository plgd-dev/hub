package commander

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
)

// Commander provides sync commands.
type Commander struct {
	subscriber eventbus.Subscriber
	raClient   service.ResourceAggregateClient
}

// NewCommander instancies commander
func NewCommander(subscriber eventbus.Subscriber, raClient service.ResourceAggregateClient) *Commander {
	return &Commander{
		subscriber: subscriber,
		raClient:   raClient,
	}
}

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

// DeleteResource sends delete resource command to resource aggregate and wait for resource deleted event from eventbus.
func (c *Commander) DeleteResource(ctx context.Context, req *commands.DeleteResourceRequest) (*events.ResourceDeleted, error) {
	h := newDeleteHandler(req.GetCorrelationId())
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), []string{req.GetResourceId().GetDeviceId()}, h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to subscriber: %w", err)
	}
	defer obs.Close()

	_, err = c.raClient.DeleteResource(ctx, req)
	if err != nil {
		return nil, err
	}

	return h.recv(ctx)
}

// UpdateResource sends update resource command to resource aggregate and wait for resource updated event from eventbus.
func (c *Commander) UpdateResource(ctx context.Context, req *commands.UpdateResourceRequest) (*events.ResourceUpdated, error) {
	h := newUpdateHandler(req.GetCorrelationId())
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), []string{req.GetResourceId().GetDeviceId()}, h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to subscriber: %w", err)
	}
	defer obs.Close()

	_, err = c.raClient.UpdateResource(ctx, req)
	if err != nil {
		return nil, err
	}

	return h.recv(ctx)
}

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

// RetrieveResource sends retrieve resource command to resource aggregate and wait for resource retrieved event from eventbus.
func (c *Commander) RetrieveResource(ctx context.Context, req *commands.RetrieveResourceRequest) (*events.ResourceRetrieved, error) {
	h := newRetrieveHandler(req.GetCorrelationId())
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), []string{req.GetResourceId().GetDeviceId()}, h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to subscriber: %w", err)
	}
	defer obs.Close()

	_, err = c.raClient.RetrieveResource(ctx, req)
	if err != nil {
		return nil, err
	}

	return h.recv(ctx)
}
