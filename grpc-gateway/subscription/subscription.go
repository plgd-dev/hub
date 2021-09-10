package subscription

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/cloud/authorization/client"
	ownerEvents "github.com/plgd-dev/cloud/authorization/events"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/kit/strings"
	"go.uber.org/atomic"
	"google.golang.org/grpc/codes"
)

type SendEventFunc = func(e *pb.Event) error
type ErrFunc = func(err error)

type deduplicateEvent struct {
	version    uint64
	validUntil *time.Time
}

type Sub struct {
	ctx                   atomic.Value
	filter                FilterBitmask
	send                  SendEventFunc
	req                   *pb.SubscribeToEvents_CreateSubscription
	id                    string
	correlationID         string
	errFunc               ErrFunc
	eventsSub             *subscriber.Subscriber
	expiration            time.Duration
	ownerSubClose         atomic.Value
	devicesEventsObserver map[string]eventbus.Observer
	filteredDeviceIDs     strings.Set
	filteredResourceIDs   strings.Set
	resourceDirectory     pb.GrpcGatewayClient
	closed                atomic.Bool
	deduplicateEvents     map[string]deduplicateEvent

	eventsCache   chan eventbus.EventUnmarshaler
	ownerCache    chan *ownerEvents.Event
	done          chan struct{}
	eventsCacheWg sync.WaitGroup
}

func isFilteredDevice(filteredDeviceIDs strings.Set, deviceID string) bool {
	if len(filteredDeviceIDs) == 0 {
		return true
	}
	return filteredDeviceIDs.HasOneOf(deviceID)
}

func isFilteredResourceIDs(filteredResourceIDs strings.Set, resourceID string) bool {
	if len(filteredResourceIDs) == 0 {
		return true
	}
	return filteredResourceIDs.HasOneOf(resourceID)
}

var eventTypeToBitmaks = map[string]FilterBitmask{
	(&events.ResourceCreatePending{}).EventType():       FilterBitmaskResourceCreatePending,
	(&events.ResourceCreated{}).EventType():             FilterBitmaskResourceCreated,
	(&events.ResourceRetrievePending{}).EventType():     FilterBitmaskResourceRetrievePending,
	(&events.ResourceRetrieved{}).EventType():           FilterBitmaskResourceRetrieved,
	(&events.ResourceUpdatePending{}).EventType():       FilterBitmaskResourceUpdatePending,
	(&events.ResourceUpdated{}).EventType():             FilterBitmaskResourceUpdated,
	(&events.ResourceDeletePending{}).EventType():       FilterBitmaskResourceDeletePending,
	(&events.ResourceDeleted{}).EventType():             FilterBitmaskResourceDeleted,
	(&events.DeviceMetadataUpdatePending{}).EventType(): FilterBitmaskDeviceMetadataUpdatePending,
	(&events.DeviceMetadataUpdated{}).EventType():       FilterBitmaskDeviceMetadataUpdated,
	(&events.ResourceChanged{}).EventType():             FilterBitmaskResourceChanged,
	(&events.ResourceLinksPublished{}).EventType():      FilterBitmaskResourcesPublished,
	(&events.ResourceLinksUnpublished{}).EventType():    FilterBitmaskResourcesUnpublished,
}

func isFilteredBit(filteredEventTypes FilterBitmask, bit FilterBitmask) bool {
	return filteredEventTypes&bit != 0
}

func isFilteredEventype(filteredEventTypes FilterBitmask, eventType string) bool {
	bit, ok := eventTypeToBitmaks[eventType]
	if !ok {
		return false
	}
	return isFilteredBit(filteredEventTypes, bit)
}

func (s *Sub) deinitDeviceLocked(deviceID string) error {
	devicesEventsObserver, ok := s.devicesEventsObserver[deviceID]
	if !ok {
		return nil
	}
	delete(s.devicesEventsObserver, deviceID)
	return devicesEventsObserver.Close()
}

type resourceEventHandler func(eventbus.EventUnmarshaler) (*pb.Event, error)

func handleResourcesPublished(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.ResourceLinksPublished
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourcePublished{
			ResourcePublished: &e,
		},
	}, nil
}

func handleResourcesUnpublished(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.ResourceLinksUnpublished
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceUnpublished{
			ResourceUnpublished: &e,
		},
	}, nil
}

func handleResourceChanged(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.ResourceChanged
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceChanged{
			ResourceChanged: &e,
		},
	}, nil
}

func handleResourceUpdatePending(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.ResourceUpdatePending
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceUpdatePending{
			ResourceUpdatePending: &e,
		},
	}, nil
}

func handleResourceUpdated(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.ResourceUpdated
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceUpdated{
			ResourceUpdated: &e,
		},
	}, nil
}

func handleResourceRetrievePending(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.ResourceRetrievePending
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceRetrievePending{
			ResourceRetrievePending: &e,
		},
	}, nil
}

func handleResourceRetrieved(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.ResourceRetrieved
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceRetrieved{
			ResourceRetrieved: &e,
		},
	}, nil
}

func handleResourceDeletePending(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.ResourceDeletePending
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceDeletePending{
			ResourceDeletePending: &e,
		},
	}, nil
}

func handleResourceDeleted(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.ResourceDeleted
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceDeleted{
			ResourceDeleted: &e,
		},
	}, nil
}

func handleResourceCreatePending(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.ResourceCreatePending
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceCreatePending{
			ResourceCreatePending: &e,
		},
	}, nil
}

func handleResourceCreated(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.ResourceCreated
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_ResourceCreated{
			ResourceCreated: &e,
		},
	}, nil
}

func handleDeviceMetadataUpdatePending(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.DeviceMetadataUpdatePending
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_DeviceMetadataUpdatePending{
			DeviceMetadataUpdatePending: &e,
		},
	}, nil
}

func handleDeviceMetadataUpdated(eu eventbus.EventUnmarshaler) (*pb.Event, error) {
	var e events.DeviceMetadataUpdated
	if err := eu.Unmarshal(&e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %v: %w", eu, err)
	}
	return &pb.Event{
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &e,
		},
	}, nil
}

var eventToHandler = map[string]resourceEventHandler{
	(&events.ResourceCreatePending{}).EventType():       handleResourceCreatePending,
	(&events.ResourceCreated{}).EventType():             handleResourceCreated,
	(&events.ResourceRetrievePending{}).EventType():     handleResourceRetrievePending,
	(&events.ResourceRetrieved{}).EventType():           handleResourceRetrieved,
	(&events.ResourceUpdatePending{}).EventType():       handleResourceUpdatePending,
	(&events.ResourceUpdated{}).EventType():             handleResourceUpdated,
	(&events.ResourceDeletePending{}).EventType():       handleResourceDeletePending,
	(&events.ResourceDeleted{}).EventType():             handleResourceDeleted,
	(&events.DeviceMetadataUpdatePending{}).EventType(): handleDeviceMetadataUpdatePending,
	(&events.DeviceMetadataUpdated{}).EventType():       handleDeviceMetadataUpdated,
	(&events.ResourceChanged{}).EventType():             handleResourceChanged,
	(&events.ResourceLinksPublished{}).EventType():      handleResourcesPublished,
	(&events.ResourceLinksUnpublished{}).EventType():    handleResourcesUnpublished,
}

func (s *Sub) handleEvent(eu eventbus.EventUnmarshaler) {
	if !s.isFiltered(eu) {
		return
	}
	handler, ok := eventToHandler[eu.EventType()]
	if !ok {
		log.Errorf("unhandled event type %v", eu.EventType())
		return
	}

	ev, err := handler(eu)
	if err != nil {
		log.Errorf("cannot get event: %w", err)
		return
	}
	ev.CorrelationId = s.correlationID
	ev.SubscriptionId = s.id
	err = s.send(ev)
	if err != nil {
		log.Errorf("cannot send event %v: %w", ev, err)
	}
}

func (s *Sub) isFiltered(ev event) bool {
	if s.closed.Load() {
		return false
	}
	return !s.isDuplicatedEvent(ev)
}

func (s *Sub) Handle(ctx context.Context, iter eventbus.Iter) (err error) {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if !isFilteredResourceIDs(s.filteredResourceIDs, eu.AggregateID()) {
			continue
		}
		if !isFilteredEventype(s.filter, eu.EventType()) {
			continue
		}

		select {
		case <-s.done:
			return nil
		case s.eventsCache <- eu:
		}
	}

	return iter.Err()
}

func (s *Sub) SetContext(ctx context.Context) {
	s.ctx.Store(ctx)
}

func (s *Sub) Context() context.Context {
	return s.ctx.Load().(context.Context)
}

func (s *Sub) Id() string {
	return s.id
}

func (s *Sub) subscribeToOwnerEvents(ownerCache *client.OwnerCache) error {
	owner, err := grpc.OwnerFromTokenMD(s.Context(), ownerCache.OwnerClaim())
	if err != nil {
		return grpc.ForwardFromError(codes.InvalidArgument, err)
	}
	close, err := ownerCache.Subscribe(owner, s.onOwnerEvent)
	if err != nil {
		return err
	}
	s.setOwnerSubClose(close)
	return nil
}

func (s *Sub) setOwnerSubClose(close func()) {
	s.ownerSubClose.Store(close)
}

func (s *Sub) closeOwnerSub() {
	c := s.ownerSubClose.Load().(func())
	c()
}

func (s *Sub) initOwnerSubscription(ownerCache *client.OwnerCache) ([]string, error) {
	err := s.subscribeToOwnerEvents(ownerCache)
	if err != nil {
		_ = s.Close()
		return nil, err
	}
	devices, err := ownerCache.GetDevices(s.Context())
	if err != nil {
		_ = s.Close()
		return nil, err
	}

	devices, err = s.initEventSubscriptions(devices)
	if err != nil {
		_ = s.Close()
		return nil, err
	}
	return devices, nil
}

func (s *Sub) start() {
	s.eventsCacheWg.Add(1)
	go func() {
		defer s.eventsCacheWg.Done()
		s.run()
	}()
}

func (s *Sub) initEvents(devices []string) error {
	if !s.req.GetIncludeCurrentState() {
		return nil
	}
	var initEventFuncs = []func(devices []string, validUntil *time.Time) error{
		s.sendDevicesRegistered,
		s.initDeviceMetadataUpdated,
		s.initResourcesPublished,
		s.initResourceChanged,
		s.initPendingCommands,
	}
	var errors []error
	var validUntil time.Time
	start := time.Now()
	for _, f := range initEventFuncs {
		err := f(devices, &validUntil)
		if err != nil {
			errors = append(errors, err)
		}
	}
	now := time.Now()
	validUntil = now.Add(now.Sub(start) + s.expiration)
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

func (s *Sub) Init(ownerCache *client.OwnerCache) error {
	devices, err := s.initOwnerSubscription(ownerCache)
	if err != nil {
		return err
	}
	err = s.initEvents(devices)
	if err != nil {
		_ = s.Close()
		return err
	}
	s.start()
	return nil
}

func (s *Sub) filterDevices(devices []string) []string {
	filteredDevices := make([]string, 0, len(devices))
	for _, d := range devices {
		if isFilteredDevice(s.filteredDeviceIDs, d) {
			filteredDevices = append(filteredDevices, d)
		}
	}
	return filteredDevices
}

func (s *Sub) initEventSubscriptions(deviceIDs []string) ([]string, error) {
	var errors []error
	filteredDevices := make([]string, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		if _, ok := s.devicesEventsObserver[deviceID]; ok {
			continue
		}
		obs, err := s.eventsSub.Subscribe(s.Context(), deviceID+"."+s.id, utils.GetDeviceSubject(deviceID), s)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		s.devicesEventsObserver[deviceID] = obs
		filteredDevices = append(filteredDevices, deviceID)
	}
	if len(errors) > 0 {
		return filteredDevices, fmt.Errorf("cannot init events subscription for devices[%v]: %v", deviceIDs, errors)
	}
	return filteredDevices, nil
}

func (s *Sub) sendDevicesRegistered(deviceIDs []string, validUntil *time.Time) error {
	if !isFilteredBit(s.filter, FilterBitmaskDeviceRegistered) {
		return nil
	}

	err := s.send(&pb.Event{
		SubscriptionId: s.id,
		CorrelationId:  s.correlationID,
		Type: &pb.Event_DeviceRegistered_{
			DeviceRegistered: &pb.Event_DeviceRegistered{
				DeviceIds: deviceIDs,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("cannot send devices registered for devices %v: %w", deviceIDs, err)
	}
	return nil
}

func (s *Sub) initResourceChanged(deviceIDs []string, validUntil *time.Time) error {
	if !isFilteredBit(s.filter, FilterBitmaskResourceChanged) {
		return nil
	}
	errFunc := func(err error) error {
		return fmt.Errorf("cannot init resources changed events for devices %v: %w", deviceIDs, err)
	}
	deviceIdFilter := deviceIDs
	if len(s.req.GetResourceIdFilter()) > 0 {
		deviceIdFilter = nil
	}
	resourcesClient, err := s.resourceDirectory.GetResources(s.Context(), &pb.GetResourcesRequest{
		DeviceIdFilter:   deviceIdFilter,
		ResourceIdFilter: s.req.GetResourceIdFilter(),
	})
	if err != nil {
		return errFunc(fmt.Errorf("cannot get resources: %w", err))
	}
	for {
		recv, err := resourcesClient.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errFunc(fmt.Errorf("cannot receive resource: %w", err))
		}
		if recv.GetData() == nil {
			// event doesn't contains data - resource is not initialized yet
			continue
		}
		ev := &pb.Event{
			SubscriptionId: s.id,
			CorrelationId:  s.correlationID,
			Type: &pb.Event_ResourceChanged{
				ResourceChanged: recv.GetData(),
			},
		}
		s.fillDeduplicateEvent(ev.GetResourceChanged(), validUntil)
		err = s.send(ev)
		if err != nil {
			return errFunc(fmt.Errorf("cannot send a resource: %w", err))
		}
	}
}

func (s *Sub) initDeviceMetadataUpdated(deviceIDs []string, validUntil *time.Time) error {
	if !isFilteredBit(s.filter, FilterBitmaskDeviceMetadataUpdated) {
		return nil
	}
	errFunc := func(err error) error {
		return fmt.Errorf("cannot init devices metadata for devices %v: %w", deviceIDs, err)
	}
	linksClient, err := s.resourceDirectory.GetDevicesMetadata(s.Context(), &pb.GetDevicesMetadataRequest{
		DeviceIdFilter: deviceIDs,
	})
	if err != nil {
		return errFunc(fmt.Errorf("cannot get devices metadata: %w", err))
	}
	for {
		recv, err := linksClient.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errFunc(fmt.Errorf("cannot receive devices metadata: %w", err))
		}
		ev := &pb.Event{
			SubscriptionId: s.id,
			CorrelationId:  s.correlationID,
			Type: &pb.Event_DeviceMetadataUpdated{
				DeviceMetadataUpdated: recv,
			},
		}
		s.fillDeduplicateEvent(ev.GetDeviceMetadataUpdated(), validUntil)
		err = s.send(ev)
		if err != nil {
			return errFunc(fmt.Errorf("cannot send a devices metadata: %w", err))
		}
	}
}

func (s *Sub) initResourcesPublished(deviceIDs []string, validUntil *time.Time) error {
	if !isFilteredBit(s.filter, FilterBitmaskResourcesPublished) {
		return nil
	}
	errFunc := func(err error) error {
		return fmt.Errorf("cannot init resources published events for devices %v: %w", deviceIDs, err)
	}
	linksClient, err := s.resourceDirectory.GetResourceLinks(s.Context(), &pb.GetResourceLinksRequest{
		DeviceIdFilter: deviceIDs,
	})
	if err != nil {
		return errFunc(fmt.Errorf("cannot get resource links: %w", err))
	}
	for {
		recv, err := linksClient.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errFunc(fmt.Errorf("cannot receive resource links: %w", err))
		}
		ev := &pb.Event{
			SubscriptionId: s.id,
			CorrelationId:  s.correlationID,
			Type: &pb.Event_ResourcePublished{
				ResourcePublished: recv,
			},
		}
		s.fillDeduplicateEvent(ev.GetResourcePublished(), validUntil)
		err = s.send(ev)
		if err != nil {
			return errFunc(fmt.Errorf("cannot send a resource links: %w", err))
		}
	}
}

func pendingCommandToEvent(cmd *pb.PendingCommand) (*pb.Event, event) {
	switch c := cmd.GetCommand().(type) {
	case *pb.PendingCommand_DeviceMetadataUpdatePending:
		return &pb.Event{
			Type: &pb.Event_DeviceMetadataUpdatePending{
				DeviceMetadataUpdatePending: c.DeviceMetadataUpdatePending,
			},
		}, c.DeviceMetadataUpdatePending
	case *pb.PendingCommand_ResourceCreatePending:
		return &pb.Event{
			Type: &pb.Event_ResourceCreatePending{
				ResourceCreatePending: c.ResourceCreatePending,
			},
		}, c.ResourceCreatePending
	case *pb.PendingCommand_ResourceDeletePending:
		return &pb.Event{
			Type: &pb.Event_ResourceDeletePending{
				ResourceDeletePending: c.ResourceDeletePending,
			},
		}, c.ResourceDeletePending
	case *pb.PendingCommand_ResourceRetrievePending:
		return &pb.Event{
			Type: &pb.Event_ResourceRetrievePending{
				ResourceRetrievePending: c.ResourceRetrievePending,
			},
		}, c.ResourceRetrievePending
	case *pb.PendingCommand_ResourceUpdatePending:
		return &pb.Event{
			Type: &pb.Event_ResourceUpdatePending{
				ResourceUpdatePending: c.ResourceUpdatePending,
			},
		}, c.ResourceUpdatePending
	}
	return nil, nil
}

type event = interface {
	EventType() string
	AggregateID() string
	Version() uint64
}

func deduplicateEventKey(ev event) string {
	return ev.AggregateID() + ev.EventType()
}

func (s *Sub) isDuplicatedEvent(ev event) bool {
	key := deduplicateEventKey(ev)
	dedupEvent, ok := s.deduplicateEvents[key]
	if !ok {
		return false
	}
	if dedupEvent.version >= ev.Version() {
		return true
	}
	return false
}

func (s *Sub) fillDeduplicateEvent(v event, validUntil *time.Time) {
	key := deduplicateEventKey(v)
	dedupEvent, ok := s.deduplicateEvents[key]
	if !ok || dedupEvent.version < v.Version() {
		s.deduplicateEvents[key] = deduplicateEvent{
			version:    v.Version(),
			validUntil: validUntil,
		}
	}
}

func (s *Sub) initPendingCommands(deviceIDs []string, validUntil *time.Time) error {
	if !isFilteredBit(s.filter,
		FilterBitmaskDeviceMetadataUpdatePending|
			FilterBitmaskResourceCreatePending|
			FilterBitmaskResourceRetrievePending|
			FilterBitmaskResourceUpdatePending|
			FilterBitmaskResourceDeletePending) {
		return nil
	}
	errFunc := func(err error) error {
		return fmt.Errorf("cannot init pending commands for devices %v: %w", deviceIDs, err)
	}

	deviceIdFilter := deviceIDs
	if len(s.req.GetResourceIdFilter()) > 0 {
		deviceIdFilter = nil
	}

	pendingCommands, err := s.resourceDirectory.GetPendingCommands(s.Context(), &pb.GetPendingCommandsRequest{
		DeviceIdFilter:   deviceIdFilter,
		ResourceIdFilter: s.req.GetResourceIdFilter(),
		CommandFilter:    BitmaskToFilterPendingsCommands(s.filter),
	})
	if err != nil {
		return errFunc(fmt.Errorf("cannot get pending commands: %w", err))
	}
	for {
		recv, err := pendingCommands.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errFunc(fmt.Errorf("cannot receive pending command: %w", err))
		}
		ev, deduplicateEvent := pendingCommandToEvent(recv)
		if ev == nil {
			s.errFunc(errFunc(fmt.Errorf("unknown recv command %T", recv.GetCommand())))
			continue
		}
		ev.CorrelationId = s.correlationID
		ev.SubscriptionId = s.id

		s.fillDeduplicateEvent(deduplicateEvent, validUntil)

		err = s.send(ev)
		if err != nil {
			return errFunc(fmt.Errorf("cannot send a pending command: %w", err))
		}
	}
}

func (s *Sub) onRegisteredEvent(e *ownerEvents.DevicesRegistered) {
	devices := s.filterDevices(e.GetDeviceIds())
	devices, err := s.initEventSubscriptions(devices)
	if err != nil {
		s.errFunc(err)
		return
	}
	if len(devices) == 0 {
		return
	}
	if !s.req.GetIncludeCurrentState() {
		var validUntil time.Time
		start := time.Now()
		err = s.sendDevicesRegistered(devices, &validUntil)
		now := time.Now()
		validUntil = now.Add(now.Sub(start) + s.expiration)
	} else {
		err = s.initEvents(devices)
	}
	if err != nil {
		s.errFunc(err)
		return
	}
}

func (s *Sub) onUnregisteredEvent(e *ownerEvents.DevicesUnregistered) {
	devices := s.filterDevices(e.GetDeviceIds())
	if len(devices) == 0 {
		return
	}
	for _, deviceID := range devices {
		err := s.deinitDeviceLocked(deviceID)
		if err != nil {
			s.errFunc(fmt.Errorf("cannot deinit device %v: %w", deviceID, err))
		}
	}
	if isFilteredBit(s.filter, s.filter&FilterBitmaskDeviceUnregistered) {
		err := s.send(&pb.Event{
			SubscriptionId: s.id,
			CorrelationId:  s.correlationID,
			Type: &pb.Event_DeviceUnregistered_{
				DeviceUnregistered: &pb.Event_DeviceUnregistered{
					DeviceIds: devices,
				},
			},
		})
		if err != nil {
			s.errFunc(fmt.Errorf("cannot send device unregistered event for devices %v: %w", devices, err))
		}
	}
}

func (s *Sub) onOwnerEvent(e *ownerEvents.Event) {
	select {
	case <-s.done:
	case s.ownerCache <- e:
	}
}

func (s *Sub) handleOwnerEvent(e *ownerEvents.Event) {
	if s.closed.Load() {
		return
	}
	switch {
	case e.GetDevicesRegistered() != nil:
		s.onRegisteredEvent(e.GetDevicesRegistered())
	case e.GetDevicesUnregistered() != nil:
		s.onUnregisteredEvent(e.GetDevicesUnregistered())
	}
}

func (s *Sub) setClosed() bool {
	return s.closed.CAS(false, true)
}

func cleanUpDevicesEventsObservers(devicesEventsObserver map[string]eventbus.Observer) error {
	var errors []error
	for _, obs := range devicesEventsObserver {
		err := obs.Close()
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

func (s *Sub) cleanUp(devicesEventsObserver map[string]eventbus.Observer) error {
	close(s.done)
	s.eventsCacheWg.Wait()
	s.closeOwnerSub()
	return cleanUpDevicesEventsObservers(devicesEventsObserver)
}

// Close closes subscription. Be carefull it cause deadlock when you call it from send function.
func (s *Sub) Close() error {
	if !s.setClosed() {
		return nil
	}
	return s.cleanUp(s.devicesEventsObserver)
}

func (s *Sub) dropExpiredDeduplicateEvents(now time.Time) {
	for key, val := range s.deduplicateEvents {
		if val.validUntil == nil || now.After(*val.validUntil) {
			delete(s.deduplicateEvents, key)
		}
	}
}

func (s *Sub) run() {
	tick := time.NewTicker(s.expiration)
	defer tick.Stop()
	dropDeduplicatesEvents := make(chan time.Time)
	close(dropDeduplicatesEvents)
	for {
		var timeC <-chan time.Time
		timeC = dropDeduplicatesEvents
		closeTime := func() bool { return true }
		if len(s.deduplicateEvents) > 0 {
			timer := time.NewTimer(time.Second)
			closeTime = timer.Stop
			timeC = timer.C
		}
		select {
		case <-s.done:
			return
		case now := <-tick.C:
			s.dropExpiredDeduplicateEvents(now)
		case event := <-s.eventsCache:
			s.handleEvent(event)
		case ownerEvent := <-s.ownerCache:
			s.handleOwnerEvent(ownerEvent)
		case <-timeC:
			closeTime()
		}
	}
}

func New(ctx context.Context, eventsSub *subscriber.Subscriber, resourceDirectory pb.GrpcGatewayClient, send SendEventFunc, correlationID string, cacheSize int, expiration time.Duration, errFunc ErrFunc, req *pb.SubscribeToEvents_CreateSubscription) *Sub {
	bitmask := EventsFilterToBitmask(req.GetEventFilter())
	filteredResourceIDs := strings.MakeSet()
	filteredDeviceIDs := strings.MakeSet(req.GetDeviceIdFilter()...)
	for _, r := range req.GetResourceIdFilter() {
		v := commands.ResourceIdFromString(r)
		if v == nil {
			continue
		}
		filteredResourceIDs.Add(v.ToUUID())
		filteredDeviceIDs.Add(v.GetDeviceId())
		if len(req.GetEventFilter()) > 0 {
			if bitmask&(FilterBitmaskDeviceMetadataUpdatePending|FilterBitmaskDeviceMetadataUpdated) != 0 {
				filteredResourceIDs.Add(commands.MakeStatusResourceUUID(v.GetDeviceId()))
			}
			if bitmask&(FilterBitmaskResourcesPublished|FilterBitmaskResourcesUnpublished) != 0 {
				filteredResourceIDs.Add(commands.MakeLinksResourceUUID(v.GetDeviceId()))
			}
		}
	}
	if expiration <= 0 {
		expiration = time.Second * 60
	}
	id := uuid.NewString()
	if errFunc == nil {
		errFunc = func(err error) {}
	} else {
		newErrFunc := func(err error) {
			errFunc(fmt.Errorf("correlationId: %v, subscriptionId: %v: %w", correlationID, id, err))
		}
		errFunc = newErrFunc
	}
	var ctxAtomic atomic.Value
	ctxAtomic.Store(ctx)

	var ctxSubClose atomic.Value
	ctxSubClose.Store(func() {})
	return &Sub{
		ctx:                   ctxAtomic,
		filter:                EventsFilterToBitmask(req.GetEventFilter()),
		send:                  send,
		req:                   req,
		id:                    id,
		eventsSub:             eventsSub,
		filteredDeviceIDs:     strings.MakeSet(req.GetDeviceIdFilter()...),
		filteredResourceIDs:   filteredResourceIDs,
		resourceDirectory:     resourceDirectory,
		errFunc:               errFunc,
		correlationID:         correlationID,
		expiration:            expiration,
		devicesEventsObserver: make(map[string]eventbus.Observer),
		deduplicateEvents:     make(map[string]deduplicateEvent),
		eventsCache:           make(chan eventbus.EventUnmarshaler, cacheSize),
		ownerCache:            make(chan *ownerEvents.Event, cacheSize),
		done:                  make(chan struct{}),
		ownerSubClose:         ctxSubClose,
	}
}
