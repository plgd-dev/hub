package client

import (
	"context"
	"errors"
	"fmt"
	"hash/crc64"
	"sync"
	"time"

	"github.com/google/uuid"
	pbGRPC "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry/propagation"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	kitSync "github.com/plgd-dev/kit/v2/sync"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type syncVersion struct {
	dbVersionWasSet   bool
	dbVersion         uint64
	natsVersion       uint64
	natsVersionWasSet bool
}

type DeviceSubscriptionHandlers struct {
	operations             Operations
	mutex                  sync.Mutex
	pendingCommandsVersion map[uint64]*syncVersion
}

type Operations interface {
	RetrieveResource(ctx context.Context, event *events.ResourceRetrievePending) error
	UpdateResource(ctx context.Context, event *events.ResourceUpdatePending) error
	DeleteResource(ctx context.Context, event *events.ResourceDeletePending) error
	CreateResource(ctx context.Context, event *events.ResourceCreatePending) error
	UpdateDeviceMetadata(ctx context.Context, event *events.DeviceMetadataUpdatePending) error
	// Fatal error occurred during reconnection to the server. Client shall call DeviceSubscriber.Close().
	OnDeviceSubscriberReconnectError(err error)
}

func NewDeviceSubscriptionHandlers(operations Operations) *DeviceSubscriptionHandlers {
	return &DeviceSubscriptionHandlers{
		operations:             operations,
		pendingCommandsVersion: make(map[uint64]*syncVersion),
	}
}

func (h *DeviceSubscriptionHandlers) wantToProcessEvent(key uint64, eventVersion uint64, fromDB bool) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	v, ok := h.pendingCommandsVersion[key]
	if !ok {
		v = new(syncVersion)
		if fromDB {
			v.dbVersion = eventVersion
			v.dbVersionWasSet = true
		} else {
			v.natsVersion = eventVersion
			v.natsVersionWasSet = true
		}
		h.pendingCommandsVersion[key] = v
		return true
	}
	if fromDB {
		if v.dbVersionWasSet && eventVersion <= v.dbVersion {
			// the order of events from the DB is guaranteed
			return false
		}
		v.dbVersion = eventVersion
		v.dbVersionWasSet = true
		if v.natsVersionWasSet && eventVersion <= v.natsVersion {
			// version from db is smaller than nats - drop it
			return false
		}
		return true
	}
	if !v.natsVersionWasSet || eventVersion > v.natsVersion {
		// update nats version
		v.natsVersion = eventVersion
		v.natsVersionWasSet = true
	}
	if v.dbVersionWasSet && eventVersion <= v.dbVersion {
		// version from nats is smaller than db - drop it
		return false
	}
	// the order of events from the nats is not guaranteed !!!
	return true
}

func toCRC64(v string) uint64 {
	c := crc64.New(crc64.MakeTable(crc64.ISO))
	c.Write([]byte(v))
	return c.Sum64()
}

func (h *DeviceSubscriptionHandlers) HandleResourceUpdatePending(ctx context.Context, val *events.ResourceUpdatePending, fromDB bool) error {
	if !h.wantToProcessEvent(toCRC64(val.GetResourceId().ToUUID().String()+val.EventType()), val.Version(), fromDB) {
		return nil
	}

	err := h.operations.UpdateResource(ctx, val)
	if err != nil {
		return fmt.Errorf("unable to update resource %v: %w", val, err)
	}
	return err
}

func (h *DeviceSubscriptionHandlers) HandleResourceRetrievePending(ctx context.Context, val *events.ResourceRetrievePending, fromDB bool) error {
	if !h.wantToProcessEvent(toCRC64(val.GetResourceId().ToUUID().String()+val.EventType()), val.Version(), fromDB) {
		return nil
	}

	err := h.operations.RetrieveResource(ctx, val)
	if err != nil {
		return fmt.Errorf("unable to retrieve resource %v: %w", val, err)
	}
	return err
}

func (h *DeviceSubscriptionHandlers) HandleResourceDeletePending(ctx context.Context, val *events.ResourceDeletePending, fromDB bool) error {
	if !h.wantToProcessEvent(toCRC64(val.GetResourceId().ToUUID().String()+val.EventType()), val.Version(), fromDB) {
		return nil
	}

	err := h.operations.DeleteResource(ctx, val)
	if err != nil {
		return fmt.Errorf("unable to delete resource %v: %w", val, err)
	}
	return err
}

func (h *DeviceSubscriptionHandlers) HandleResourceCreatePending(ctx context.Context, val *events.ResourceCreatePending, fromDB bool) error {
	if !h.wantToProcessEvent(toCRC64(val.GetResourceId().ToUUID().String()+val.EventType()), val.Version(), fromDB) {
		return nil
	}

	err := h.operations.CreateResource(ctx, val)
	if err != nil {
		return fmt.Errorf("unable to create resource %v: %w", val, err)
	}
	return err
}

func (h *DeviceSubscriptionHandlers) HandleDeviceMetadataUpdatePending(ctx context.Context, val *events.DeviceMetadataUpdatePending, fromDB bool) error {
	if !h.wantToProcessEvent(toCRC64(val.AggregateID()+val.EventType()), val.Version(), fromDB) {
		return nil
	}

	err := h.operations.UpdateDeviceMetadata(ctx, val)
	if err != nil {
		return fmt.Errorf("unable to update device metadata %v: %w", val, err)
	}
	return err
}

type setHandlerRequest struct {
	ctx context.Context
	h   *DeviceSubscriptionHandlers
}

type DeviceSubscriber struct {
	rdClient pbGRPC.GrpcGatewayClient
	deviceID string
	done     chan struct{}

	pendingCommandsVersion *kitSync.Map
	observer               eventbus.Observer

	mutex                  sync.Mutex
	pendingCommandsHandler *DeviceSubscriptionHandlers
	reconnectChan          chan bool
	setHandlerChan         chan *setHandlerRequest
	closeFunc              func()
	factoryRetry           func() RetryFunc
	getContext             func() (context.Context, context.CancelFunc)
	tracerProvider         trace.TracerProvider
}

type RetryFunc = func() (when time.Time, err error)

func NewDeviceSubscriber(getContext func() (context.Context, context.CancelFunc), owner, deviceID string, factoryRetry func() RetryFunc, rdClient pbGRPC.GrpcGatewayClient, resourceSubscriber *subscriber.Subscriber, tracerProvider trace.TracerProvider) (*DeviceSubscriber, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	s := DeviceSubscriber{
		deviceID:               deviceID,
		rdClient:               rdClient,
		pendingCommandsVersion: kitSync.NewMap(),
		reconnectChan:          make(chan bool, 1),
		setHandlerChan:         make(chan *setHandlerRequest, 1),
		done:                   make(chan struct{}),
		factoryRetry:           factoryRetry,
		getContext:             getContext,
		tracerProvider:         tracerProvider,
	}
	ctx, cancel := getContext()
	defer cancel()

	observer, err := resourceSubscriber.Subscribe(ctx, uuid.String(), utils.GetDeviceSubject(owner, deviceID), &s)
	if err != nil {
		return nil, err
	}
	s.observer = observer
	reconnectID := resourceSubscriber.AddReconnectFunc(s.triggerReconnect)
	var wg sync.WaitGroup
	s.closeFunc = func() {
		wg.Wait()
		resourceSubscriber.RemoveReconnectFunc(reconnectID)
		s.mutex.Lock()
		defer s.mutex.Unlock()
		s.pendingCommandsHandler = nil
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.reconnect()
	}()
	return &s, nil
}

func (s *DeviceSubscriber) tryReconnectToGRPC(getContext func() (context.Context, context.CancelFunc), wantToSetPendingCommandsHandler bool, pendingCommandsHandler *DeviceSubscriptionHandlers) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if wantToSetPendingCommandsHandler {
		s.pendingCommandsHandler = pendingCommandsHandler
	}

	if s.pendingCommandsHandler == nil {
		return false
	}

	err := s.coldStartPendingCommandsLocked(getContext, s.pendingCommandsHandler)
	if err != nil {
		var grpcStatus interface{ GRPCStatus() *status.Status }
		code := codes.Unknown
		if errors.As(err, &grpcStatus) {
			code = grpcStatus.GRPCStatus().Code()
		}
		if code != codes.Unavailable {
			s.pendingCommandsHandler.operations.OnDeviceSubscriberReconnectError(err)
			return false
		}
		return false
	}
	return true
}

func (s *DeviceSubscriber) triggerReconnect() {
	select {
	case <-s.done:
	case s.reconnectChan <- true:
	default:
	}
}

func (s *DeviceSubscriber) reconnect() {
	for {
		var wantToSetPendingCommandsHandler bool
		var pendingCommandsHandler *DeviceSubscriptionHandlers
		var setHandlerReq *setHandlerRequest
		getContext := s.getContext
		select {
		case setHandlerReq = <-s.setHandlerChan:
			wantToSetPendingCommandsHandler = true
			pendingCommandsHandler = setHandlerReq.h
			runOnce := atomic.NewBool(true)
			getContext = func() (context.Context, context.CancelFunc) {
				ctx, cancel := s.getContext()
				if runOnce.CompareAndSwap(true, false) {
					span := trace.SpanFromContext(setHandlerReq.ctx)
					return trace.ContextWithSpan(ctx, span), cancel
				}
				return ctx, cancel
			}
		case <-s.reconnectChan:
		case <-s.done:
			return
		}
		nextRetry := s.factoryRetry()
	LOOP_TRY_RECONNECT_TO_GRPC:
		for !s.tryReconnectToGRPC(getContext, wantToSetPendingCommandsHandler, pendingCommandsHandler) {
			when, err := nextRetry()
			if err != nil {
				s.pendingCommandsHandler.operations.OnDeviceSubscriberReconnectError(err)
				return
			}
			select {
			case <-s.reconnectChan:
				break LOOP_TRY_RECONNECT_TO_GRPC
			case <-s.done:
				return
			case <-time.After(time.Until(when)):
			}
		}
	}
}

func (s *DeviceSubscriber) Close() (err error) {
	close(s.done)
	s.closeFunc()
	if s.observer == nil {
		return nil
	}
	return s.observer.Close()
}

func (s *DeviceSubscriber) createSpanEvent(ctx context.Context, name string) (context.Context, trace.Span) {
	tracer := s.tracerProvider.Tracer(
		opentelemetry.InstrumentationName,
		trace.WithInstrumentationVersion(opentelemetry.SemVersion()),
	)
	return tracer.Start(ctx, name, trace.WithSpanKind(trace.SpanKindConsumer))
}

func (s *DeviceSubscriber) processPendingCommand(ctx context.Context, h *DeviceSubscriptionHandlers, ev *pbGRPC.PendingCommand) error {
	var sendEvent func(ctx context.Context) error
	switch {
	case ev.GetResourceCreatePending() != nil:
		sendEvent = func(ctx context.Context) error {
			ctx, span := s.createSpanEvent(ctx, "CreateResource")
			defer span.End()
			return h.HandleResourceCreatePending(ctx, ev.GetResourceCreatePending(), true)
		}
	case ev.GetResourceRetrievePending() != nil:
		sendEvent = func(ctx context.Context) error {
			ctx, span := s.createSpanEvent(ctx, "RetrieveResource")
			defer span.End()
			return h.HandleResourceRetrievePending(ctx, ev.GetResourceRetrievePending(), true)
		}
	case ev.GetResourceUpdatePending() != nil:
		sendEvent = func(ctx context.Context) error {
			ctx, span := s.createSpanEvent(ctx, "UpdateResource")
			defer span.End()
			return h.HandleResourceUpdatePending(ctx, ev.GetResourceUpdatePending(), true)
		}
	case ev.GetResourceDeletePending() != nil:
		sendEvent = func(ctx context.Context) error {
			ctx, span := s.createSpanEvent(ctx, "DeleteResource")
			defer span.End()
			return h.HandleResourceDeletePending(ctx, ev.GetResourceDeletePending(), true)
		}
	case ev.GetDeviceMetadataUpdatePending() != nil:
		sendEvent = func(ctx context.Context) error {
			ctx, span := s.createSpanEvent(ctx, "UpdateDeviceMetadata")
			defer span.End()
			return h.HandleDeviceMetadataUpdatePending(ctx, ev.GetDeviceMetadataUpdatePending(), true)
		}
	}
	if sendEvent == nil {
		return nil
	}

	return sendEvent(ctx)
}

func (s *DeviceSubscriber) coldStartPendingCommandsLocked(getContext func() (context.Context, context.CancelFunc), h *DeviceSubscriptionHandlers) error {
	ctx, cancel := getContext()
	defer cancel()
	resp, err := s.rdClient.GetPendingCommands(ctx, &pbGRPC.GetPendingCommandsRequest{
		DeviceIdFilter: []string{s.deviceID},
	})
	iter := grpc.NewIterator(resp, err)
	for {
		var pendingCommand pbGRPC.PendingCommand
		if !iter.Next(&pendingCommand) {
			break
		}

		err := s.processPendingCommand(ctx, h, &pendingCommand)
		if err != nil {
			iter.Err = err
			break
		}
	}
	if iter.Err != nil {
		status, ok := status.FromError(iter.Err)
		if !ok || status.Code() != codes.NotFound {
			return fmt.Errorf("cannot retrieve pending commands for %v: %w", s.deviceID, iter.Err)
		}
	}
	return nil
}

func (s *DeviceSubscriber) SubscribeToPendingCommands(ctx context.Context, h *DeviceSubscriptionHandlers) {
	select {
	case <-s.done:
	case s.setHandlerChan <- &setHandlerRequest{
		ctx: ctx,
		h:   h,
	}:
	default:
	}
}

func (s *DeviceSubscriber) getPendingCommandsHandler() *DeviceSubscriptionHandlers {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.pendingCommandsHandler
}

func sendEvent(ctx context.Context, h *DeviceSubscriptionHandlers, ev eventbus.EventUnmarshaler) error {
	switch ev.EventType() {
	case (&events.ResourceRetrievePending{}).EventType():
		var event events.ResourceRetrievePending
		err := ev.Unmarshal(&event)
		if err != nil {
			return fmt.Errorf("cannot unmarshal resource retrieve pending event('%v'): %w", ev, err)
		}
		return h.HandleResourceRetrievePending(propagation.CtxWithTrace(ctx, event.GetOpenTelemetryCarrier()), &event, false)
	case (&events.ResourceUpdatePending{}).EventType():
		var event events.ResourceUpdatePending
		err := ev.Unmarshal(&event)
		if err != nil {
			return fmt.Errorf("cannot unmarshal resource update pending event('%v'): %w", ev, err)
		}
		return h.HandleResourceUpdatePending(propagation.CtxWithTrace(ctx, event.GetOpenTelemetryCarrier()), &event, false)
	case (&events.ResourceDeletePending{}).EventType():
		var event events.ResourceDeletePending
		err := ev.Unmarshal(&event)
		if err != nil {
			return fmt.Errorf("cannot unmarshal resource delete pending event('%v'): %w", ev, err)
		}
		return h.HandleResourceDeletePending(propagation.CtxWithTrace(ctx, event.GetOpenTelemetryCarrier()), &event, false)
	case (&events.ResourceCreatePending{}).EventType():
		var event events.ResourceCreatePending
		err := ev.Unmarshal(&event)
		if err != nil {
			return fmt.Errorf("cannot unmarshal resource create pending event('%v'): %w", ev, err)
		}
		return h.HandleResourceCreatePending(propagation.CtxWithTrace(ctx, event.GetOpenTelemetryCarrier()), &event, false)
	case (&events.DeviceMetadataUpdatePending{}).EventType():
		var event events.DeviceMetadataUpdatePending
		err := ev.Unmarshal(&event)
		if err != nil {
			return fmt.Errorf("cannot unmarshal device metadate update pending event('%v'): %w", ev, err)
		}
		return h.HandleDeviceMetadataUpdatePending(propagation.CtxWithTrace(ctx, event.GetOpenTelemetryCarrier()), &event, false)
	}
	return nil
}

func (s *DeviceSubscriber) Handle(ctx context.Context, iter eventbus.Iter) (err error) {
	pendingCommandsHandler := s.getPendingCommandsHandler()
	if pendingCommandsHandler == nil {
		return nil
	}
	for {
		ev, ok := iter.Next(ctx)
		if !ok {
			break
		}
		err := sendEvent(ctx, pendingCommandsHandler, ev)
		if err != nil {
			return err
		}
	}
	return nil
}
