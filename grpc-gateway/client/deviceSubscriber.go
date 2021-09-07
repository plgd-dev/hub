package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	kitSync "github.com/plgd-dev/kit/sync"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type syncVersion struct {
	sync.Mutex
	value uint64
}

type DeviceSubscriptionHandlers struct {
	operations             Operations
	pendingCommandsVersion *kitSync.Map
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
		pendingCommandsVersion: kitSync.NewMap(),
	}
}

func (h *DeviceSubscriptionHandlers) wantToProcessEvent(key string, eventVersion uint64) bool {
	valI, loaded := h.pendingCommandsVersion.LoadOrStoreWithFunc(key, func(value interface{}) interface{} {
		val := value.(*syncVersion)
		val.Lock()
		return val
	}, func() interface{} {
		newVal := syncVersion{
			value: eventVersion,
		}
		newVal.Lock()
		return &newVal
	})
	version := valI.(*syncVersion)
	defer version.Unlock()
	if loaded {
		if eventVersion < version.value {
			return false
		}
		version.value = eventVersion
	}
	return true
}

func (h *DeviceSubscriptionHandlers) HandleResourceUpdatePending(ctx context.Context, val *events.ResourceUpdatePending) error {
	if !h.wantToProcessEvent(val.GetResourceId().ToUUID()+val.EventType(), val.Version()) {
		return nil
	}

	err := h.operations.UpdateResource(ctx, val)
	if err != nil {
		return fmt.Errorf("unable to update resource %v: %w", val, err)
	}
	return err
}

func (h *DeviceSubscriptionHandlers) HandleResourceRetrievePending(ctx context.Context, val *events.ResourceRetrievePending) error {
	if !h.wantToProcessEvent(val.GetResourceId().ToUUID()+val.EventType(), val.Version()) {
		return nil
	}

	err := h.operations.RetrieveResource(ctx, val)
	if err != nil {
		return fmt.Errorf("unable to retrieve resource %v: %w", val, err)
	}
	return err
}

func (h *DeviceSubscriptionHandlers) HandleResourceDeletePending(ctx context.Context, val *events.ResourceDeletePending) error {
	if !h.wantToProcessEvent(val.GetResourceId().ToUUID()+val.EventType(), val.Version()) {
		return nil
	}

	err := h.operations.DeleteResource(ctx, val)
	if err != nil {
		return fmt.Errorf("unable to delete resource %v: %w", val, err)
	}
	return err
}

func (h *DeviceSubscriptionHandlers) HandleResourceCreatePending(ctx context.Context, val *events.ResourceCreatePending) error {
	if !h.wantToProcessEvent(val.GetResourceId().ToUUID()+val.EventType(), val.Version()) {
		return nil
	}

	err := h.operations.CreateResource(ctx, val)
	if err != nil {
		return fmt.Errorf("unable to create resource %v: %w", val, err)
	}
	return err
}

func (h *DeviceSubscriptionHandlers) HandleDeviceMetadataUpdatePending(ctx context.Context, val *events.DeviceMetadataUpdatePending) error {
	if !h.wantToProcessEvent(val.AggregateID()+val.EventType(), val.Version()) {
		return nil
	}

	err := h.operations.UpdateDeviceMetadata(ctx, val)
	if err != nil {
		return fmt.Errorf("unable to update device metadata %v: %w", val, err)
	}
	return err
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
	closeFunc              func()
	factoryRetry           func() RetryFunc
	getContext             func() (context.Context, context.CancelFunc)
}

type RetryFunc = func() (when time.Time, err error)

func NewDeviceSubscriber(getContext func() (context.Context, context.CancelFunc), deviceID string, factoryRetry func() RetryFunc, rdClient pbGRPC.GrpcGatewayClient, resourceSubscriber *subscriber.Subscriber) (*DeviceSubscriber, error) {
	uuid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	s := DeviceSubscriber{
		deviceID:               deviceID,
		rdClient:               rdClient,
		pendingCommandsVersion: kitSync.NewMap(),
		reconnectChan:          make(chan bool, 1),
		done:                   make(chan struct{}),
		factoryRetry:           factoryRetry,
		getContext:             getContext,
	}
	ctx, cancel := getContext()
	defer cancel()

	observer, err := resourceSubscriber.Subscribe(ctx, uuid.String(), utils.GetDeviceSubject(deviceID), &s)
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

func (s *DeviceSubscriber) tryReconnectToGRPC() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.pendingCommandsHandler == nil {
		return false
	}

	err := s.coldStartPendingCommandsLocked(s.pendingCommandsHandler)
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
		select {
		case <-s.reconnectChan:
		case <-s.done:
			return
		}
		nextRetry := s.factoryRetry()
	LOOP_TRY_RECONNECT_TO_GRPC:
		for !s.tryReconnectToGRPC() {
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

func (s *DeviceSubscriber) processPendingCommand(ctx context.Context, h *DeviceSubscriptionHandlers, ev *pbGRPC.PendingCommand) error {
	var sendEvent func(ctx context.Context) error
	switch {
	case ev.GetResourceCreatePending() != nil:
		sendEvent = func(ctx context.Context) error {
			return h.HandleResourceCreatePending(ctx, ev.GetResourceCreatePending())
		}
	case ev.GetResourceRetrievePending() != nil:
		sendEvent = func(ctx context.Context) error {
			return h.HandleResourceRetrievePending(ctx, ev.GetResourceRetrievePending())
		}
	case ev.GetResourceUpdatePending() != nil:
		sendEvent = func(ctx context.Context) error {
			return h.HandleResourceUpdatePending(ctx, ev.GetResourceUpdatePending())
		}
	case ev.GetResourceDeletePending() != nil:
		sendEvent = func(ctx context.Context) error {
			return h.HandleResourceDeletePending(ctx, ev.GetResourceDeletePending())
		}
	case ev.GetDeviceMetadataUpdatePending() != nil:
		sendEvent = func(ctx context.Context) error {
			return h.HandleDeviceMetadataUpdatePending(ctx, ev.GetDeviceMetadataUpdatePending())
		}
	}
	if sendEvent == nil {
		return nil
	}

	return sendEvent(ctx)
}

func (s *DeviceSubscriber) coldStartPendingCommandsLocked(h *DeviceSubscriptionHandlers) error {
	ctx, cancel := s.getContext()
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

func (s *DeviceSubscriber) SubscribeToPendingCommands(h *DeviceSubscriptionHandlers) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.pendingCommandsHandler = h
	s.triggerReconnect()
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
		return h.HandleResourceRetrievePending(ctx, &event)
	case (&events.ResourceUpdatePending{}).EventType():
		var event events.ResourceUpdatePending
		err := ev.Unmarshal(&event)
		if err != nil {
			return fmt.Errorf("cannot unmarshal resource update pending event('%v'): %w", ev, err)
		}
		return h.HandleResourceUpdatePending(ctx, &event)
	case (&events.ResourceDeletePending{}).EventType():
		var event events.ResourceDeletePending
		err := ev.Unmarshal(&event)
		if err != nil {
			return fmt.Errorf("cannot unmarshal resource delete pending event('%v'): %w", ev, err)
		}
		return h.HandleResourceDeletePending(ctx, &event)
	case (&events.ResourceCreatePending{}).EventType():
		var event events.ResourceCreatePending
		err := ev.Unmarshal(&event)
		if err != nil {
			return fmt.Errorf("cannot unmarshal resource create pending event('%v'): %w", ev, err)
		}
		return h.HandleResourceCreatePending(ctx, &event)
	case (&events.DeviceMetadataUpdatePending{}).EventType():
		var event events.DeviceMetadataUpdatePending
		err := ev.Unmarshal(&event)
		if err != nil {
			return fmt.Errorf("cannot unmarshal device metadate update pending event('%v'): %w", ev, err)
		}
		return h.HandleDeviceMetadataUpdatePending(ctx, &event)
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
