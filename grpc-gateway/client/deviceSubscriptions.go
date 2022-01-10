package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	kitSync "github.com/plgd-dev/kit/v2/sync"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

// ResourcePublishedHandler handler of events.
type ResourcePublishedHandler = interface {
	HandleResourcePublished(ctx context.Context, val *events.ResourceLinksPublished) error
}

// ResourceUnpublishedHandler handler of events.
type ResourceUnpublishedHandler = interface {
	HandleResourceUnpublished(ctx context.Context, val *events.ResourceLinksUnpublished) error
}

// ResourceUpdatePendingHandler handler of events
type ResourceUpdatePendingHandler = interface {
	HandleResourceUpdatePending(ctx context.Context, val *events.ResourceUpdatePending) error
}

// ResourceUpdatedHandler handler of events
type ResourceUpdatedHandler = interface {
	HandleResourceUpdated(ctx context.Context, val *events.ResourceUpdated) error
}

// ResourceRetrievePendingHandler handler of events
type ResourceRetrievePendingHandler = interface {
	HandleResourceRetrievePending(ctx context.Context, val *events.ResourceRetrievePending) error
}

// ResourceRetrievedHandler handler of events
type ResourceRetrievedHandler = interface {
	HandleResourceRetrieved(ctx context.Context, val *events.ResourceRetrieved) error
}

// ResourceDeletePendingHandler handler of events
type ResourceDeletePendingHandler = interface {
	HandleResourceDeletePending(ctx context.Context, val *events.ResourceDeletePending) error
}

// ResourceDeletedHandler handler of events
type ResourceDeletedHandler = interface {
	HandleResourceDeleted(ctx context.Context, val *events.ResourceDeleted) error
}

// ResourceCreatePendingHandler handler of events
type ResourceCreatePendingHandler = interface {
	HandleResourceCreatePending(ctx context.Context, val *events.ResourceCreatePending) error
}

// ResourceCreatedHandler handler of events
type ResourceCreatedHandler = interface {
	HandleResourceCreated(ctx context.Context, val *events.ResourceCreated) error
}

func NewDeviceSubscriptions(ctx context.Context, gwClient pb.GrpcGatewayClient, errFunc func(err error)) (*DeviceSubscriptions, error) {
	client, err := New(gwClient).SubscribeToEventsWithCurrentState(ctx, time.Minute)
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

func (s *DeviceSubscriptions) send(req *pb.SubscribeToEvents) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.client.Send(req)
}

func (s *DeviceSubscriptions) doOp(ctx context.Context, req *pb.SubscribeToEvents) (*pb.Event, error) {
	subscriptionIDChan := make(chan *pb.Event, 1)
	s.processedOperations.Store(req.GetCorrelationId(), subscriptionIDChan)
	defer s.processedOperations.Delete(req.GetCorrelationId())

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
	client              pb.GrpcGateway_SubscribeToEventsClient
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
	ResourceCreatePendingHandler
	ResourceCreatedHandler
}

func (s *deviceSub) HandleResourcePublished(ctx context.Context, val *events.ResourceLinksPublished) error {
	if s.ResourcePublishedHandler == nil {
		return fmt.Errorf("ResourcePublishedHandler in not supported")
	}
	return s.ResourcePublishedHandler.HandleResourcePublished(ctx, val)
}

func (s *deviceSub) HandleResourceUnpublished(ctx context.Context, val *events.ResourceLinksUnpublished) error {
	if s.ResourceUnpublishedHandler == nil {
		return fmt.Errorf("ResourceUnpublishedHandler in not supported")
	}
	return s.ResourceUnpublishedHandler.HandleResourceUnpublished(ctx, val)
}

func (s *deviceSub) HandleResourceUpdatePending(ctx context.Context, val *events.ResourceUpdatePending) error {
	if s.ResourceUpdatePendingHandler == nil {
		return fmt.Errorf("ResourceUpdatePendingHandler in not supported")
	}
	return s.ResourceUpdatePendingHandler.HandleResourceUpdatePending(ctx, val)
}

func (s *deviceSub) HandleResourceUpdated(ctx context.Context, val *events.ResourceUpdated) error {
	if s.ResourceUpdatedHandler == nil {
		return fmt.Errorf("ResourceUpdatedHandler in not supported")
	}
	return s.ResourceUpdatedHandler.HandleResourceUpdated(ctx, val)
}

func (s *deviceSub) HandleResourceRetrievePending(ctx context.Context, val *events.ResourceRetrievePending) error {
	if s.ResourceRetrievePendingHandler == nil {
		return fmt.Errorf("ResourceRetrievePendingHandler in not supported")
	}
	return s.ResourceRetrievePendingHandler.HandleResourceRetrievePending(ctx, val)
}

func (s *deviceSub) HandleResourceRetrieved(ctx context.Context, val *events.ResourceRetrieved) error {
	if s.ResourceRetrievedHandler == nil {
		return fmt.Errorf("ResourceRetrievedHandler in not supported")
	}
	return s.ResourceRetrievedHandler.HandleResourceRetrieved(ctx, val)
}

func (s *deviceSub) HandleResourceDeletePending(ctx context.Context, val *events.ResourceDeletePending) error {
	if s.ResourceDeletePendingHandler == nil {
		return fmt.Errorf("ResourceDeletePendingHandler in not supported")
	}
	return s.ResourceDeletePendingHandler.HandleResourceDeletePending(ctx, val)
}

func (s *deviceSub) HandleResourceDeleted(ctx context.Context, val *events.ResourceDeleted) error {
	if s.ResourceDeletedHandler == nil {
		return fmt.Errorf("ResourceDeletedHandler in not supported")
	}
	return s.ResourceDeletedHandler.HandleResourceDeleted(ctx, val)
}

func (s *deviceSub) HandleResourceCreatePending(ctx context.Context, val *events.ResourceCreatePending) error {
	if s.ResourceCreatePendingHandler == nil {
		return fmt.Errorf("ResourceCreatePendingHandler in not supported")
	}
	return s.ResourceCreatePendingHandler.HandleResourceCreatePending(ctx, val)
}

func (s *deviceSub) HandleResourceCreated(ctx context.Context, val *events.ResourceCreated) error {
	if s.ResourceCreatedHandler == nil {
		return fmt.Errorf("ResourceCreatedHandler in not supported")
	}
	return s.ResourceCreatedHandler.HandleResourceCreated(ctx, val)
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

func getSubscribeTypeAndHandler(closeErrorHandler SubscriptionHandler, handle interface{}) ([]pb.SubscribeToEvents_CreateSubscription_Event, *deviceSub, error) {
	if closeErrorHandler == nil {
		return nil, nil, fmt.Errorf("invalid closeErrorHandler")
	}
	var resourcePublishedHandler ResourcePublishedHandler
	var resourceUnpublishedHandler ResourceUnpublishedHandler
	var resourceUpdatePendingHandler ResourceUpdatePendingHandler
	var resourceUpdatedHandler ResourceUpdatedHandler
	var resourceRetrievePendingHandler ResourceRetrievePendingHandler
	var resourceRetrievedHandler ResourceRetrievedHandler
	var resourceDeletePendingHandler ResourceDeletePendingHandler
	var resourceDeletedHandler ResourceDeletedHandler
	var resourceCreatePendingHandler ResourceCreatePendingHandler
	var resourceCreatedHandler ResourceCreatedHandler

	filterEvents := make([]pb.SubscribeToEvents_CreateSubscription_Event, 0, 2)
	if v, ok := handle.(ResourcePublishedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_RESOURCE_PUBLISHED)
		resourcePublishedHandler = v
	}
	if v, ok := handle.(ResourceUnpublishedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_RESOURCE_UNPUBLISHED)
		resourceUnpublishedHandler = v
	}
	if v, ok := handle.(ResourceUpdatePendingHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING)
		resourceUpdatePendingHandler = v
	}
	if v, ok := handle.(ResourceUpdatedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATED)
		resourceUpdatedHandler = v
	}
	if v, ok := handle.(ResourceRetrievePendingHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING)
		resourceRetrievePendingHandler = v
	}
	if v, ok := handle.(ResourceRetrievedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVED)
		resourceRetrievedHandler = v
	}
	if v, ok := handle.(ResourceDeletePendingHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETE_PENDING)
		resourceDeletePendingHandler = v
	}
	if v, ok := handle.(ResourceDeletedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETED)
		resourceDeletedHandler = v
	}
	if v, ok := handle.(ResourceCreatePendingHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATE_PENDING)
		resourceCreatePendingHandler = v
	}
	if v, ok := handle.(ResourceCreatedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATED)
		resourceCreatedHandler = v
	}

	if len(filterEvents) == 0 {
		return nil, nil, fmt.Errorf("invalid handler - supported handlers: ResourcePublishedHandler, ResourceUnpublishedHandler, ResourceUpdatePendingHandler, ResourceUpdatedHandler, ResourceRetrievePendingHandler, ResourceRetrievedHandler, ResourceDeletePendingHandler, ResourceDeletedHandler, ResourceCreatePendingHandler, ResourceCreatedHandler")
	}

	return filterEvents, &deviceSub{
		SubscriptionHandler:            closeErrorHandler,
		ResourcePublishedHandler:       resourcePublishedHandler,
		ResourceUnpublishedHandler:     resourceUnpublishedHandler,
		ResourceUpdatePendingHandler:   resourceUpdatePendingHandler,
		ResourceUpdatedHandler:         resourceUpdatedHandler,
		ResourceRetrievePendingHandler: resourceRetrievePendingHandler,
		ResourceRetrievedHandler:       resourceRetrievedHandler,
		ResourceDeletePendingHandler:   resourceDeletePendingHandler,
		ResourceDeletedHandler:         resourceDeletedHandler,
		ResourceCreatePendingHandler:   resourceCreatePendingHandler,
		ResourceCreatedHandler:         resourceCreatedHandler,
	}, nil
}

func (s *DeviceSubscriptions) Subscribe(ctx context.Context, deviceID string, closeErrorHandler SubscriptionHandler, handle interface{}) (*Subcription, error) {
	token, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("cannot generate token for subscribe")
	}

	filterEvents, eh, err := getSubscribeTypeAndHandler(closeErrorHandler, handle)
	if err != nil {
		return nil, err
	}

	s.handlers.Store(token.String(), eh)

	ev, err := s.doOp(ctx, &pb.SubscribeToEvents{
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				DeviceIdFilter: []string{deviceID},
				EventFilter:    filterEvents,
			},
		},
		CorrelationId: token.String(),
	})
	if err != nil {
		return nil, err
	}

	var cancelled uint32
	cancel := func(ctx context.Context) error {
		if !atomic.CompareAndSwapUint32(&cancelled, 0, 1) {
			return nil
		}
		cancelToken, err := uuid.NewRandom()
		if err != nil {
			return fmt.Errorf("cannot generate token for cancellation")
		}
		_, err = s.doOp(ctx, &pb.SubscribeToEvents{
			Action: &pb.SubscribeToEvents_CancelSubscription_{
				CancelSubscription: &pb.SubscribeToEvents_CancelSubscription{
					SubscriptionId: ev.GetSubscriptionId(),
				},
			},
			CorrelationId: cancelToken.String(),
		})
		s.handlers.Delete(token.String())
		return err
	}

	return &Subcription{
		id:     ev.GetSubscriptionId(),
		cancel: cancel,
	}, nil
}

func (s *DeviceSubscriptions) closeSend() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.client.CloseSend()
}

// Cancel cancels subscription.
func (s *DeviceSubscriptions) Cancel() (wait func(), err error) {
	if !atomic.CompareAndSwapUint32(&s.canceled, 0, 1) {
		return s.wait, nil
	}

	err = s.closeSend()
	if err != nil {
		return nil, err
	}
	return s.wait, nil
}

func (s *DeviceSubscriptions) cancelSubscription(subID string) error {
	return s.send(&pb.SubscribeToEvents{
		Action: &pb.SubscribeToEvents_CancelSubscription_{
			CancelSubscription: &pb.SubscribeToEvents_CancelSubscription{
				SubscriptionId: subID,
			},
		},
	})
}

func (s *DeviceSubscriptions) getHandler(ev *pb.Event) *deviceSub {
	ha, ok := s.handlers.Load(ev.GetCorrelationId())
	if !ok {
		s.errFunc(fmt.Errorf("invalid event from subscription - ID: %v, CorrelationId: %v, Type %T", ev.GetSubscriptionId(), ev.GetCorrelationId(), ev.GetType()))
		err := s.send(&pb.SubscribeToEvents{
			Action: &pb.SubscribeToEvents_CancelSubscription_{
				CancelSubscription: &pb.SubscribeToEvents_CancelSubscription{
					SubscriptionId: ev.GetSubscriptionId(),
				},
			},
		})
		if err != nil {
			s.errFunc(fmt.Errorf("cannot cancel subscription - ID: %v, CorrelationId: %v, Type %T: %w", ev.GetSubscriptionId(), ev.GetCorrelationId(), ev.GetType(), err))
		}
		return nil
	}
	return ha.(*deviceSub)
}

func (s *DeviceSubscriptions) handleRecvError(err error) {
	if _, err2 := s.Cancel(); err2 != nil {
		s.errFunc(fmt.Errorf("failed to cancel device subscription: %w", err2))
	}
	cancelled := false
	if err == io.EOF {
		cancelled = atomic.LoadUint32(&s.canceled) > 0
	}
	for _, h := range s.handlers.PullOutAll() {
		if cancelled {
			h.(SubscriptionHandler).OnClose()
		} else {
			h.(SubscriptionHandler).Error(err)
		}
	}
}

func (s *DeviceSubscriptions) handleSubscriptionCanceled(e *pb.Event) (processed bool) {
	cancel := e.GetSubscriptionCanceled()
	if cancel == nil {
		return false
	}

	reason := cancel.GetReason()
	h, ok := s.handlers.PullOut(e.GetCorrelationId())
	if !ok {
		return true
	}
	ha := h.(*deviceSub)
	if reason == "" {
		ha.OnClose()
		return true
	}
	ha.Error(fmt.Errorf(reason))
	return true
}

func (s *DeviceSubscriptions) handleOperationProcessed(e *pb.Event) (processed bool) {
	operationProcessed := e.GetOperationProcessed()
	if operationProcessed == nil {
		return false
	}

	opChan, ok := s.processedOperations.Load(e.GetCorrelationId())
	if !ok {
		return true
	}
	select {
	case opChan.(chan *pb.Event) <- e:
	default:
		s.errFunc(fmt.Errorf("cannot send operation processed - ID: %v, CorrelationId: %v, Type %T: channel is exhausted",
			e.GetSubscriptionId(), e.GetCorrelationId(), e.GetType()))
	}

	return true
}

func (s *DeviceSubscriptions) handleEventByType(e *pb.Event, h *deviceSub) (ok bool) {
	if ct := e.GetResourcePublished(); ct != nil {
		err := h.HandleResourcePublished(s.client.Context(), ct)
		return err == nil
	}
	if ct := e.GetResourceUnpublished(); ct != nil {
		err := h.HandleResourceUnpublished(s.client.Context(), ct)
		return err == nil
	}
	if ct := e.GetResourceUpdatePending(); ct != nil {
		err := h.HandleResourceUpdatePending(s.client.Context(), ct)
		return err == nil
	}
	if ct := e.GetResourceUpdated(); ct != nil {
		err := h.HandleResourceUpdated(s.client.Context(), ct)
		return err == nil
	}
	if ct := e.GetResourceRetrievePending(); ct != nil {
		err := h.HandleResourceRetrievePending(s.client.Context(), ct)
		return err == nil
	}
	if ct := e.GetResourceRetrieved(); ct != nil {
		err := h.HandleResourceRetrieved(s.client.Context(), ct)
		return err == nil
	}
	if ct := e.GetResourceDeletePending(); ct != nil {
		err := h.HandleResourceDeletePending(s.client.Context(), ct)
		return err == nil
	}
	if ct := e.GetResourceDeleted(); ct != nil {
		err := h.HandleResourceDeleted(s.client.Context(), ct)
		return err == nil
	}
	if ct := e.GetResourceCreatePending(); ct != nil {
		err := h.HandleResourceCreatePending(s.client.Context(), ct)
		return err == nil
	}
	if ct := e.GetResourceCreated(); ct != nil {
		err := h.HandleResourceCreated(s.client.Context(), ct)
		return err == nil
	}

	handler, ok := s.handlers.PullOut(e.GetCorrelationId())
	if !ok {
		return true
	}
	ha := handler.(*deviceSub)
	ha.Error(fmt.Errorf("unknown event %T occurs on recv device events: %+v", e, e))
	return false
}

func (s *DeviceSubscriptions) handleEvent(e *pb.Event) {
	h := s.getHandler(e)
	if h == nil {
		return
	}

	if s.handleEventByType(e, h) {
		return
	}

	err := s.cancelSubscription(e.GetSubscriptionId())
	if err != nil {
		s.errFunc(fmt.Errorf("cannot cancel subscription - ID: %v, CorrelationId: %v, Type %T: %w",
			e.GetSubscriptionId(), e.GetCorrelationId(), e.GetType(), err))
	}
}

func (s *DeviceSubscriptions) runRecv() {
	for {
		ev, err := s.client.Recv()
		if err != nil {
			s.handleRecvError(err)
			return
		}

		if s.handleSubscriptionCanceled(ev) {
			continue
		}

		if s.handleOperationProcessed(ev) {
			continue
		}

		s.handleEvent(ev)
	}
}
