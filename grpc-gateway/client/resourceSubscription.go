package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

// SubscriptionHandler handler of events.
type SubscriptionHandler = interface {
	OnClose()
	Error(err error)
}

// ResourceContentChangedHandler handler of events.
type ResourceContentChangedHandler = interface {
	HandleResourceContentChanged(ctx context.Context, val *events.ResourceChanged) error
}

// ResourceSubscription subscription.
type ResourceSubscription struct {
	client                        pb.GrpcGateway_SubscribeToEventsClient
	subscriptionID                string
	closeErrorHandler             SubscriptionHandler
	resourceContentChangedHandler ResourceContentChangedHandler

	wait     func()
	canceled uint32
}

// NewResourceSubscription creates new resource content changed subscription.
// JWT token must be stored in context for grpc call.
func (c *Client) NewResourceSubscription(ctx context.Context, resourceID *commands.ResourceId, handle SubscriptionHandler) (*ResourceSubscription, error) {
	return NewResourceSubscription(ctx, resourceID, handle, handle, c.gateway)
}

// NewResourceSubscription creates new resource content changed subscription.
// JWT token must be stored in context for grpc call.
func NewResourceSubscription(ctx context.Context, resourceID *commands.ResourceId, closeErrorHandler SubscriptionHandler, handle interface{}, gwClient pb.GrpcGatewayClient) (*ResourceSubscription, error) {
	var resourceContentChangedHandler ResourceContentChangedHandler
	filterEvents := make([]pb.SubscribeToEvents_CreateSubscription_Event, 0, 1)
	if v, ok := handle.(ResourceContentChangedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED)
		resourceContentChangedHandler = v
	}

	if resourceContentChangedHandler == nil {
		return nil, errors.New("invalid handler - it's supports: ResourceContentChangedHandler")
	}
	client, err := New(gwClient).SubscribeToEventsWithCurrentState(ctx, time.Minute)
	if err != nil {
		return nil, err
	}

	err = client.Send(&pb.SubscribeToEvents{
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				ResourceIdFilter: []*pb.ResourceIdFilter{
					{
						ResourceId: resourceID,
					},
				},
				EventFilter: filterEvents,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	ev, err := client.Recv()
	if err != nil {
		return nil, err
	}
	op := ev.GetOperationProcessed()
	if op == nil {
		return nil, fmt.Errorf("unexpected event %+v", ev)
	}
	if op.GetErrorStatus().GetCode() != pb.Event_OperationProcessed_ErrorStatus_OK {
		return nil, errors.New(op.GetErrorStatus().GetMessage())
	}

	var wg sync.WaitGroup
	sub := &ResourceSubscription{
		client:                        client,
		closeErrorHandler:             closeErrorHandler,
		subscriptionID:                ev.GetSubscriptionId(),
		resourceContentChangedHandler: resourceContentChangedHandler,
		wait:                          wg.Wait,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		sub.runRecv()
	}()

	return sub, nil
}

// Cancel cancels subscription.
func (s *ResourceSubscription) Cancel() (wait func(), err error) {
	if !atomic.CompareAndSwapUint32(&s.canceled, 0, 1) {
		return s.wait, nil
	}
	err = s.client.CloseSend()
	if err != nil {
		return nil, err
	}
	return s.wait, nil
}

// ID returns subscription id.
func (s *ResourceSubscription) ID() string {
	return s.subscriptionID
}

func (s *ResourceSubscription) cancelAndHandleError(err error) {
	var errors *multierror.Error
	if err != nil {
		errors = multierror.Append(errors, err)
	}
	if _, err := s.Cancel(); err != nil {
		errors = multierror.Append(errors, fmt.Errorf("failed to cancel resource subscription: %w", err))
	}
	if errors.ErrorOrNil() != nil {
		s.closeErrorHandler.Error(errors)
	}
}

func (s *ResourceSubscription) handleCancel(cancel *pb.Event_SubscriptionCanceled) {
	reason := cancel.GetReason()
	if reason == "" {
		s.closeErrorHandler.OnClose()
		return
	}
	s.closeErrorHandler.Error(errors.New(reason))
}

func (s *ResourceSubscription) runRecv() {
	for {
		ev, err := s.client.Recv()
		if errors.Is(err, io.EOF) {
			s.cancelAndHandleError(nil)
			s.closeErrorHandler.OnClose()
			return
		}
		if err != nil {
			s.cancelAndHandleError(err)
			return
		}
		cancel := ev.GetSubscriptionCanceled()
		if cancel != nil {
			s.cancelAndHandleError(nil)
			s.handleCancel(cancel)
			return
		}

		ct := ev.GetResourceChanged()
		if ct == nil {
			s.cancelAndHandleError(fmt.Errorf("unknown event occurs %T on recv resource events: %+v", ev, ev))
			return
		}
		err = s.resourceContentChangedHandler.HandleResourceContentChanged(s.client.Context(), ct)
		if err != nil {
			s.cancelAndHandleError(err)
			return
		}
	}
}

func ToResourceSubscription(v interface{}, ok bool) (*ResourceSubscription, bool) {
	if !ok {
		return nil, false
	}
	if v == nil {
		return nil, false
	}
	s, ok := v.(*ResourceSubscription)
	return s, ok
}
