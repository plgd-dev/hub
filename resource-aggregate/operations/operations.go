package operations

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
)

// Operator makes commands synchronously. It makes request to resource aggregate
// and it waits for the result from the device, which is arrives in the confirm event.
type Operator struct {
	subscriber eventbus.Subscriber
	raClient   service.ResourceAggregateClient
}

// NewOperator instancies Operator
func NewOperator(subscriber eventbus.Subscriber, raClient service.ResourceAggregateClient) *Operator {
	return &Operator{
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
func (c *Operator) DeleteResource(ctx context.Context, req *commands.DeleteResourceRequest) (*events.ResourceDeleted, error) {
	h := newDeleteHandler(req.GetCorrelationId())
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), utils.GetTopics(req.GetResourceId().GetDeviceId()), h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to eventbus: %w", err)
	}
	defer obs.Close()

	_, err = c.raClient.DeleteResource(ctx, req)
	if err != nil {
		return nil, err
	}

	return h.recv(ctx)
}

// UpdateResource sends update resource command to resource aggregate and wait for resource updated event from eventbus.
func (c *Operator) UpdateResource(ctx context.Context, req *commands.UpdateResourceRequest) (*events.ResourceUpdated, error) {
	h := newUpdateHandler(req.GetCorrelationId())
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), utils.GetTopics(req.GetResourceId().GetDeviceId()), h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to eventbus: %w", err)
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
func (c *Operator) RetrieveResource(ctx context.Context, req *commands.RetrieveResourceRequest) (*events.ResourceRetrieved, error) {
	h := newRetrieveHandler(req.GetCorrelationId())
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), utils.GetTopics(req.GetResourceId().GetDeviceId()), h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to eventbus: %w", err)
	}
	defer obs.Close()

	_, err = c.raClient.RetrieveResource(ctx, req)
	if err != nil {
		return nil, err
	}

	return h.recv(ctx)
}

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

// CreateResource sends create resource command to resource aggregate and wait for resource created event from eventbus.
func (c *Operator) CreateResource(ctx context.Context, req *commands.CreateResourceRequest) (*events.ResourceRetrieved, error) {
	h := newRetrieveHandler(req.GetCorrelationId())
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), utils.GetTopics(req.GetResourceId().GetDeviceId()), h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to eventbus: %w", err)
	}
	defer obs.Close()

	_, err = c.raClient.CreateResource(ctx, req)
	if err != nil {
		return nil, err
	}

	return h.recv(ctx)
}
