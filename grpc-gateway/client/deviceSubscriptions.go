package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/gofrs/uuid"
	kitSync "github.com/plgd-dev/kit/sync"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
)

// ResourcePublishedHandler handler of events.
type ResourcePublishedHandler = interface {
	HandleResourcePublished(ctx context.Context, val *pb.Event_ResourcePublished) error
}

// ResourceUnpublishedHandler handler of events.
type ResourceUnpublishedHandler = interface {
	HandleResourceUnpublished(ctx context.Context, val *pb.Event_ResourceUnpublished) error
}

// ResourceUpdatePendingHandler handler of events
type ResourceUpdatePendingHandler = interface {
	HandleResourceUpdatePending(ctx context.Context, val *pb.Event_ResourceUpdatePending) error
}

// ResourceUpdatedHandler handler of events
type ResourceUpdatedHandler = interface {
	HandleResourceUpdated(ctx context.Context, val *pb.Event_ResourceUpdated) error
}

// ResourceRetrievePendingHandler handler of events
type ResourceRetrievePendingHandler = interface {
	HandleResourceRetrievePending(ctx context.Context, val *pb.Event_ResourceRetrievePending) error
}

// ResourceRetrievedHandler handler of events
type ResourceRetrievedHandler = interface {
	HandleResourceRetrieved(ctx context.Context, val *pb.Event_ResourceRetrieved) error
}

// ResourceDeletePendingHandler handler of events
type ResourceDeletePendingHandler = interface {
	HandleResourceDeletePending(ctx context.Context, val *pb.Event_ResourceDeletePending) error
}

// ResourceDeletedHandler handler of events
type ResourceDeletedHandler = interface {
	HandleResourceDeleted(ctx context.Context, val *pb.Event_ResourceDeleted) error
}

func NewDeviceSubscriptions(ctx context.Context, gwClient pb.GrpcGatewayClient, errFunc func(err error)) (*DeviceSubscriptions, error) {
	client, err := gwClient.SubscribeForEvents(ctx)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	subs := DeviceSubscriptions{
		client:              client,
		processedOperations: kitSync.NewMap(),
		handlers:            kitSync.NewMap(),
		wait:                wg.Wait,
		errFunc:             errFunc,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		subs.runRecv()
	}()
	return &subs, nil
}

func (s *DeviceSubscriptions) send(req *pb.SubscribeForEvents) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.client.Send(req)
}

func (s *DeviceSubscriptions) doOp(ctx context.Context, req *pb.SubscribeForEvents) (*pb.Event, error) {
	subscriptionIDChan := make(chan *pb.Event, 1)
	s.processedOperations.Store(req.GetToken(), subscriptionIDChan)
	defer s.processedOperations.Delete(req.GetToken())

	err := s.send(req)
	if err != nil {
		return nil, err
	}
	var ev *pb.Event
	select {
	case ev = <-subscriptionIDChan:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	op := ev.GetOperationProcessed()
	if op == nil {
		return nil, fmt.Errorf("unexpected event %+v", ev)
	}
	if op.GetErrorStatus().GetCode() != pb.Event_OperationProcessed_ErrorStatus_OK {
		return nil, fmt.Errorf(op.GetErrorStatus().GetMessage())
	}
	return ev, nil
}

type DeviceSubscriptions struct {
	processedOperations *kitSync.Map
	handlers            *kitSync.Map
	client              pb.GrpcGateway_SubscribeForEventsClient
	mutex               sync.Mutex
	errFunc             func(err error)

	wait     func()
	canceled uint32
}

type deviceSub struct {
	SubscriptionHandler
	ResourcePublishedHandler
	ResourceUnpublishedHandler
	ResourceUpdatePendingHandler
	ResourceUpdatedHandler
	ResourceRetrievePendingHandler
	ResourceRetrievedHandler
	ResourceDeletePendingHandler
	ResourceDeletedHandler
}

func (s *deviceSub) HandleResourcePublished(ctx context.Context, val *pb.Event_ResourcePublished) error {
	if s.ResourcePublishedHandler == nil {
		return fmt.Errorf("ResourcePublishedHandler in not supported")
	}
	return s.ResourcePublishedHandler.HandleResourcePublished(ctx, val)
}

func (s *deviceSub) HandleResourceUnpublished(ctx context.Context, val *pb.Event_ResourceUnpublished) error {
	if s.ResourceUnpublishedHandler == nil {
		return fmt.Errorf("ResourceUnpublishedHandler in not supported")
	}
	return s.ResourceUnpublishedHandler.HandleResourceUnpublished(ctx, val)
}

func (s *deviceSub) HandleResourceUpdatePending(ctx context.Context, val *pb.Event_ResourceUpdatePending) error {
	if s.ResourceUpdatePendingHandler == nil {
		return fmt.Errorf("ResourceUpdatePendingHandler in not supported")
	}
	return s.ResourceUpdatePendingHandler.HandleResourceUpdatePending(ctx, val)
}

func (s *deviceSub) HandleResourceUpdated(ctx context.Context, val *pb.Event_ResourceUpdated) error {
	if s.ResourceUpdatedHandler == nil {
		return fmt.Errorf("ResourceUpdatedHandler in not supported")
	}
	return s.ResourceUpdatedHandler.HandleResourceUpdated(ctx, val)
}

func (s *deviceSub) HandleResourceRetrievePending(ctx context.Context, val *pb.Event_ResourceRetrievePending) error {
	if s.ResourceRetrievePendingHandler == nil {
		return fmt.Errorf("ResourceRetrievePendingHandler in not supported")
	}
	return s.ResourceRetrievePendingHandler.HandleResourceRetrievePending(ctx, val)
}

func (s *deviceSub) HandleResourceRetrieved(ctx context.Context, val *pb.Event_ResourceRetrieved) error {
	if s.ResourceRetrievedHandler == nil {
		return fmt.Errorf("ResourceRetrievedHandler in not supported")
	}
	return s.ResourceRetrievedHandler.HandleResourceRetrieved(ctx, val)
}

func (s *deviceSub) HandleResourceDeletePending(ctx context.Context, val *pb.Event_ResourceDeletePending) error {
	if s.ResourceDeletePendingHandler == nil {
		return fmt.Errorf("ResourceDeletePendingHandler in not supported")
	}
	return s.ResourceDeletePendingHandler.HandleResourceDeletePending(ctx, val)
}

func (s *deviceSub) HandleResourceDeleted(ctx context.Context, val *pb.Event_ResourceDeleted) error {
	if s.ResourceDeletedHandler == nil {
		return fmt.Errorf("ResourceDeletedHandler in not supported")
	}
	return s.ResourceDeletedHandler.HandleResourceDeleted(ctx, val)
}

type Subcription struct {
	id     string
	cancel func(context.Context) error
}

func (s *Subcription) ID() string {
	return s.id
}

func (s *Subcription) Cancel(ctx context.Context) error {
	return s.cancel(ctx)
}

func (s *DeviceSubscriptions) Subscribe(ctx context.Context, deviceID string, closeErrorHandler SubscriptionHandler, handle interface{}) (*Subcription, error) {
	if closeErrorHandler == nil {
		return nil, fmt.Errorf("invalid closeErrorHandler")
	}
	var resourcePublishedHandler ResourcePublishedHandler
	var resourceUnpublishedHandler ResourceUnpublishedHandler
	var resourceUpdatePendingHandler ResourceUpdatePendingHandler
	var resourceUpdatedHandler ResourceUpdatedHandler
	var resourceRetrievePendingHandler ResourceRetrievePendingHandler
	var resourceRetrievedHandler ResourceRetrievedHandler
	var resourceDeletePendingHandler ResourceDeletePendingHandler
	var resourceDeletedHandler ResourceDeletedHandler
	filterEvents := make([]pb.SubscribeForEvents_DeviceEventFilter_Event, 0, 1)
	if v, ok := handle.(ResourcePublishedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_PUBLISHED)
		resourcePublishedHandler = v
	}
	if v, ok := handle.(ResourceUnpublishedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UNPUBLISHED)
		resourceUnpublishedHandler = v
	}
	if v, ok := handle.(ResourceUpdatePendingHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATE_PENDING)
		resourceUpdatePendingHandler = v
	}
	if v, ok := handle.(ResourceUpdatedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATED)
		resourceUpdatedHandler = v
	}
	if v, ok := handle.(ResourceRetrievePendingHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVE_PENDING)
		resourceRetrievePendingHandler = v
	}
	if v, ok := handle.(ResourceRetrievedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVED)
		resourceRetrievedHandler = v
	}
	if v, ok := handle.(ResourceDeletePendingHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_DELETE_PENDING)
		resourceDeletePendingHandler = v
	}
	if v, ok := handle.(ResourceDeletedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_DELETED)
		resourceDeletedHandler = v
	}

	if resourcePublishedHandler == nil &&
		resourceUnpublishedHandler == nil &&
		resourceUpdatePendingHandler == nil &&
		resourceUpdatedHandler == nil &&
		resourceRetrievePendingHandler == nil &&
		resourceRetrievedHandler == nil &&
		resourceDeletePendingHandler == nil &&
		resourceDeletedHandler == nil {
		return nil, fmt.Errorf("invalid handler - it's supports: ResourcePublishedHandler, ResourceUnpublishedHandler, ResourceUpdatePendingHandler, ResourceUpdatedHandler, ResourceRetrievePendingHandler, ResourceRetrievedHandler, ResourceDeletePendingHandler, ResourceDeletedHandler")
	}

	token, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("cannot generate token for subscribe")
	}

	s.handlers.Store(token.String(), &deviceSub{
		SubscriptionHandler:            closeErrorHandler,
		ResourcePublishedHandler:       resourcePublishedHandler,
		ResourceUnpublishedHandler:     resourceUnpublishedHandler,
		ResourceUpdatePendingHandler:   resourceUpdatePendingHandler,
		ResourceUpdatedHandler:         resourceUpdatedHandler,
		ResourceRetrievePendingHandler: resourceRetrievePendingHandler,
		ResourceRetrievedHandler:       resourceRetrievedHandler,
		ResourceDeletePendingHandler:   resourceDeletePendingHandler,
		ResourceDeletedHandler:         resourceDeletedHandler,
	})

	ev, err := s.doOp(ctx, &pb.SubscribeForEvents{
		FilterBy: &pb.SubscribeForEvents_DeviceEvent{
			DeviceEvent: &pb.SubscribeForEvents_DeviceEventFilter{
				DeviceId:     deviceID,
				FilterEvents: filterEvents,
			},
		},
		Token: token.String(),
	})
	if err != nil {
		return nil, err
	}

	var cancelled uint32
	cancel := func(ctx context.Context) error {
		if !atomic.CompareAndSwapUint32(&cancelled, 0, 1) {
			return nil
		}
		cancelToken, err := uuid.NewV4()
		if err != nil {
			return fmt.Errorf("cannot generate token for cancellation")
		}
		_, err = s.doOp(ctx, &pb.SubscribeForEvents{
			FilterBy: &pb.SubscribeForEvents_CancelSubscription_{
				CancelSubscription: &pb.SubscribeForEvents_CancelSubscription{
					SubscriptionId: ev.GetSubscriptionId(),
				},
			},
			Token: cancelToken.String(),
		})
		s.handlers.Delete(token.String())
		return err
	}

	return &Subcription{
		id:     ev.GetSubscriptionId(),
		cancel: cancel,
	}, nil
}

// Cancel cancels subscription.
func (s *DeviceSubscriptions) Cancel() (wait func(), err error) {
	if !atomic.CompareAndSwapUint32(&s.canceled, 0, 1) {
		return s.wait, nil
	}
	err = s.client.CloseSend()
	if err != nil {
		return nil, err
	}
	return s.wait, nil
}

func (s *DeviceSubscriptions) cancelSubscription(subID string) error {
	return s.send(&pb.SubscribeForEvents{
		FilterBy: &pb.SubscribeForEvents_CancelSubscription_{
			CancelSubscription: &pb.SubscribeForEvents_CancelSubscription{
				SubscriptionId: subID,
			},
		},
	})
}

func (s *DeviceSubscriptions) getHandler(ev *pb.Event) *deviceSub {
	ha, ok := s.handlers.Load(ev.GetToken())
	if !ok {
		s.errFunc(fmt.Errorf("invalid event from subscription - ID: %v, Token: %v, Type %T", ev.GetSubscriptionId(), ev.GetToken(), ev.GetType()))
		err := s.send(&pb.SubscribeForEvents{
			FilterBy: &pb.SubscribeForEvents_CancelSubscription_{
				CancelSubscription: &pb.SubscribeForEvents_CancelSubscription{
					SubscriptionId: ev.GetSubscriptionId(),
				},
			},
		})
		if err != nil {
			s.errFunc(fmt.Errorf("cannot cancel subscription - ID: %v, Token: %v, Type %T: %w", ev.GetSubscriptionId(), ev.GetToken(), ev.GetType(), err))
		}
		return nil
	}
	return ha.(*deviceSub)
}

func (s *DeviceSubscriptions) runRecv() {
	for {
		ev, err := s.client.Recv()
		if err == io.EOF {
			s.Cancel()
			s.handlers.PullOutAll()
			for _, h := range s.handlers.PullOutAll() {
				h.(SubscriptionHandler).OnClose()
			}

			return
		}
		if err != nil {
			s.Cancel()
			s.handlers.PullOutAll()
			for _, h := range s.handlers.PullOutAll() {
				h.(SubscriptionHandler).Error(err)
			}
			return
		}
		cancel := ev.GetSubscriptionCanceled()
		if cancel != nil {
			s.Cancel()
			reason := cancel.GetReason()
			h, ok := s.handlers.PullOut(ev.GetToken())
			if !ok {
				continue
			}
			ha := h.(*deviceSub)
			if reason == "" {
				ha.OnClose()
				continue
			}
			ha.Error(fmt.Errorf(reason))
			continue
		}
		operationProcessed := ev.GetOperationProcessed()
		if operationProcessed != nil {
			opChan, ok := s.processedOperations.Load(ev.GetToken())
			if !ok {
				continue
			}
			select {
			case opChan.(chan *pb.Event) <- ev:
			default:
				s.errFunc(fmt.Errorf("cannot send operation processed - ID: %v, Token: %v, Type %T: channel is exhausted", ev.GetSubscriptionId(), ev.GetToken(), ev.GetType()))
			}
			continue
		}

		h := s.getHandler(ev)
		if h == nil {
			continue
		}
		if ct := ev.GetResourcePublished(); ct != nil {
			err = h.HandleResourcePublished(s.client.Context(), ct)
			if err == nil {
				continue
			}
			err := s.cancelSubscription(ev.GetSubscriptionId())
			if err != nil {
				s.errFunc(fmt.Errorf("cannot cancel subscription - ID: %v, Token: %v, Type %T: %w", ev.GetSubscriptionId(), ev.GetToken(), ev.GetType(), err))
			}
		} else if ct := ev.GetResourceUnpublished(); ct != nil {
			err = h.HandleResourceUnpublished(s.client.Context(), ct)
			if err == nil {
				continue
			}
			err := s.cancelSubscription(ev.GetSubscriptionId())
			if err != nil {
				s.errFunc(fmt.Errorf("cannot cancel subscription - ID: %v, Token: %v, Type %T: %w", ev.GetSubscriptionId(), ev.GetToken(), ev.GetType(), err))
			}
		} else if ct := ev.GetResourceUpdatePending(); ct != nil {
			err = h.HandleResourceUpdatePending(s.client.Context(), ct)
			if err == nil {
				continue
			}
			err := s.cancelSubscription(ev.GetSubscriptionId())
			if err != nil {
				s.errFunc(fmt.Errorf("cannot cancel subscription - ID: %v, Token: %v, Type %T: %w", ev.GetSubscriptionId(), ev.GetToken(), ev.GetType(), err))
			}
		} else if ct := ev.GetResourceUpdated(); ct != nil {
			err = h.HandleResourceUpdated(s.client.Context(), ct)
			if err == nil {
				continue
			}
			err := s.cancelSubscription(ev.GetSubscriptionId())
			if err != nil {
				s.errFunc(fmt.Errorf("cannot cancel subscription - ID: %v, Token: %v, Type %T: %w", ev.GetSubscriptionId(), ev.GetToken(), ev.GetType(), err))
			}
		} else if ct := ev.GetResourceRetrievePending(); ct != nil {
			err = h.HandleResourceRetrievePending(s.client.Context(), ct)
			if err == nil {
				continue
			}
			err := s.cancelSubscription(ev.GetSubscriptionId())
			if err != nil {
				s.errFunc(fmt.Errorf("cannot cancel subscription - ID: %v, Token: %v, Type %T: %w", ev.GetSubscriptionId(), ev.GetToken(), ev.GetType(), err))
			}
		} else if ct := ev.GetResourceRetrieved(); ct != nil {
			err = h.HandleResourceRetrieved(s.client.Context(), ct)
			if err == nil {
				continue
			}
			err := s.cancelSubscription(ev.GetSubscriptionId())
			if err != nil {
				s.errFunc(fmt.Errorf("cannot cancel subscription - ID: %v, Token: %v, Type %T: %w", ev.GetSubscriptionId(), ev.GetToken(), ev.GetType(), err))
			}
		} else if ct := ev.GetResourceDeletePending(); ct != nil {
			err = h.HandleResourceDeletePending(s.client.Context(), ct)
			if err == nil {
				continue
			}
			err := s.cancelSubscription(ev.GetSubscriptionId())
			if err != nil {
				s.errFunc(fmt.Errorf("cannot cancel subscription - ID: %v, Token: %v, Type %T: %w", ev.GetSubscriptionId(), ev.GetToken(), ev.GetType(), err))
			}
		} else if ct := ev.GetResourceDeleted(); ct != nil {
			err = h.HandleResourceDeleted(s.client.Context(), ct)
			if err == nil {
				continue
			}
			err := s.cancelSubscription(ev.GetSubscriptionId())
			if err != nil {
				s.errFunc(fmt.Errorf("cannot cancel subscription - ID: %v, Token: %v, Type %T: %w", ev.GetSubscriptionId(), ev.GetToken(), ev.GetType(), err))
			}
		} else {
			h, ok := s.handlers.PullOut(ev.GetToken())
			if !ok {
				continue
			}
			ha := h.(*deviceSub)
			ha.Error(fmt.Errorf("unknown event %T occurs on recv device events: %+v", ev, ev))
			err := s.cancelSubscription(ev.GetSubscriptionId())
			if err != nil {
				s.errFunc(fmt.Errorf("cannot cancel subscription - ID: %v, Token: %v, Type %T: %w", ev.GetSubscriptionId(), ev.GetToken(), ev.GetType(), err))
			}
		}
	}
}

func ToDeviceSubscriptions(v interface{}, ok bool) (*DeviceSubscriptions, bool) {
	if !ok {
		return nil, false
	}
	if v == nil {
		return nil, false
	}
	s, ok := v.(*DeviceSubscriptions)
	return s, ok
}
