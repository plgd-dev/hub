package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"google.golang.org/grpc/codes"
)

type createHandler struct {
	correlationID string
	encoder       func(ec *commands.Content) (*commands.Content, error)
	res           chan *events.ResourceCreated
}

func newCreateHandler(correlationID string, encoder func(ec *commands.Content) (*commands.Content, error)) *createHandler {
	return &createHandler{
		correlationID: correlationID,
		res:           make(chan *events.ResourceCreated, 1),
		encoder:       encoder,
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
		var err error
		if h.encoder != nil {
			v.Content, err = h.encoder(v.GetContent())
		}
		return v, err
	}
}

// SyncCreateResource sends create resource command to resource aggregate and wait for resource created event from eventbus.
func (c *Client) SyncCreateResource(ctx context.Context, req *commands.CreateResourceRequest) (*events.ResourceCreated, error) {
	responseContentEncoder, err := commands.GetContentEncoder(grpc.AcceptContentFromMD(ctx))
	if err != nil {
		return nil, grpc.ForwardErrorf(codes.InvalidArgument, "%v", err)
	}
	h := newCreateHandler(req.GetCorrelationId(), responseContentEncoder)
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), utils.GetTopics(req.GetResourceId().GetDeviceId()), h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to eventbus: %w", err)
	}
	defer obs.Close()

	_, err = c.CreateResource(ctx, req)
	if err != nil {
		return nil, err
	}

	return h.recv(ctx)
}
