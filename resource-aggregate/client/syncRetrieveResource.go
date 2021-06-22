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

type retrieveHandler struct {
	correlationID string
	res           chan *events.ResourceRetrieved
	encoder       func(ec *commands.Content) (*commands.Content, error)
}

func newRetrieveHandler(correlationID string, encoder func(ec *commands.Content) (*commands.Content, error)) *retrieveHandler {
	return &retrieveHandler{
		correlationID: correlationID,
		res:           make(chan *events.ResourceRetrieved, 1),
		encoder:       encoder,
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
		var err error
		if h.encoder != nil {
			v.Content, err = h.encoder(v.GetContent())
		}
		return v, err
	}
}

// SyncRetrieveResource sends retrieve resource command to resource aggregate and wait for resource retrieved event from eventbus.
func (c *Client) SyncRetrieveResource(ctx context.Context, req *commands.RetrieveResourceRequest) (*events.ResourceRetrieved, error) {
	responseContentEncoder, err := commands.GetContentEncoder(grpc.AcceptContentFromMD(ctx))
	if err != nil {
		return nil, grpc.ForwardErrorf(codes.InvalidArgument, "%v", err)
	}
	h := newRetrieveHandler(req.GetCorrelationId(), responseContentEncoder)
	obs, err := c.subscriber.Subscribe(ctx, req.GetCorrelationId(), utils.GetTopics(req.GetResourceId().GetDeviceId()), h)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to eventbus: %w", err)
	}
	defer obs.Close()

	_, err = c.RetrieveResource(ctx, req)
	if err != nil {
		return nil, err
	}

	return h.recv(ctx)
}
