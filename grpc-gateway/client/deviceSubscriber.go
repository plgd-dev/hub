package client

import (
	"context"
	"fmt"
	"sync"

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
		return fmt.Errorf("update resource %v: %w", val, err)
	}
	return err
}

func (h *DeviceSubscriptionHandlers) HandleResourceRetrievePending(ctx context.Context, val *events.ResourceRetrievePending) error {
	if !h.wantToProcessEvent(val.GetResourceId().ToUUID()+val.EventType(), val.Version()) {
		return nil
	}

	err := h.operations.RetrieveResource(ctx, val)
	if err != nil {
		return fmt.Errorf("retrieve resource %v: %w", val, err)
	}
	return err
}

func (h *DeviceSubscriptionHandlers) HandleResourceDeletePending(ctx context.Context, val *events.ResourceDeletePending) error {
	if !h.wantToProcessEvent(val.GetResourceId().ToUUID()+val.EventType(), val.Version()) {
		return nil
	}

	err := h.operations.DeleteResource(ctx, val)
	if err != nil {
		return fmt.Errorf("delete resource %v: %w", val, err)
	}
	return err
}

func (h *DeviceSubscriptionHandlers) HandleResourceCreatePending(ctx context.Context, val *events.ResourceCreatePending) error {
	if !h.wantToProcessEvent(val.GetResourceId().ToUUID()+val.EventType(), val.Version()) {
		return nil
	}

	err := h.operations.CreateResource(ctx, val)
	if err != nil {
		return fmt.Errorf("delete resource %v: %w", val, err)
	}
	return err
}

type PendingCommandsHandler interface {
	HandleResourceUpdatePending(ctx context.Context, val *events.ResourceUpdatePending) error
	HandleResourceRetrievePending(ctx context.Context, val *events.ResourceRetrievePending) error
	HandleResourceDeletePending(ctx context.Context, val *events.ResourceDeletePending) error
	HandleResourceCreatePending(ctx context.Context, val *events.ResourceCreatePending) error
}

type DeviceSubscriber struct {
	rdClient pbGRPC.GrpcGatewayClient
	deviceID string

	pendingCommandsVersion *kitSync.Map
	observer               eventbus.Observer

	mutex                  sync.Mutex
	pendingCommandsHandler PendingCommandsHandler
}

func NewDeviceSubscriber(ctx context.Context, deviceID string, rdClient pbGRPC.GrpcGatewayClient, resourceSubscriber *subscriber.Subscriber) (*DeviceSubscriber, error) {
	uuid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	s := DeviceSubscriber{
		deviceID:               deviceID,
		rdClient:               rdClient,
		pendingCommandsVersion: kitSync.NewMap(),
	}
	observer, err := resourceSubscriber.Subscribe(ctx, uuid.String(), utils.GetTopics(deviceID), &s)
	if err != nil {
		return nil, err
	}
	s.observer = observer
	return &s, nil
}

func (s *DeviceSubscriber) Close() (err error) {
	if s.observer == nil {
		return nil
	}
	return s.observer.Close()
}

func (s *DeviceSubscriber) processPendingCommand(ctx context.Context, h PendingCommandsHandler, ev *pbGRPC.PendingCommand) error {

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
	}
	if sendEvent == nil {
		return nil
	}

	return sendEvent(ctx)
}

func (s *DeviceSubscriber) SubscribeToPendingCommands(ctx context.Context, h *DeviceSubscriptionHandlers) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	resp, err := s.rdClient.RetrievePendingCommands(ctx, &pbGRPC.RetrievePendingCommandsRequest{
		DeviceIdsFilter: []string{s.deviceID},
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
	s.pendingCommandsHandler = h

	return nil
}

func (s *DeviceSubscriber) getPendingCommandsHandler() PendingCommandsHandler {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.pendingCommandsHandler
}

func sendEvent(ctx context.Context, h PendingCommandsHandler, ev eventbus.EventUnmarshaler) error {
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
