package subscription

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/identity-store/client"
	ownerEvents "github.com/plgd-dev/cloud/identity-store/events"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/kit/strings"
	"go.uber.org/atomic"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
)

type SendEventFunc = func(e *pb.Event) error
type ErrFunc = func(err error)

type Sub struct {
	ctx                   atomic.Value
	filter                FilterBitmask
	send                  SendEventFunc
	req                   *pb.SubscribeToEvents_CreateSubscription
	id                    string
	correlationID         string
	errFunc               ErrFunc
	eventsSub             *subscriber.Subscriber
	ownerSubClose         atomic.Value
	devicesEventsObserver map[string]eventbus.Observer
	filteredDeviceIDs     strings.Set
	filteredResourceIDs   strings.Set

	eventsCache   chan eventbus.EventUnmarshaler
	ownerCache    chan *ownerEvents.Event
	doneCtx       context.Context
	cancelDoneCtx context.CancelFunc
	syncGoroutine *semaphore.Weighted
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

func isFilteredEventype(filteredEventTypes FilterBitmask, eventType string) bool {
	bit, ok := eventTypeToBitmaks[eventType]
	if !ok {
		return false
	}
	return IsFilteredBit(filteredEventTypes, bit)
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
		case <-s.doneCtx.Done():
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

func (s *Sub) CorrelationId() string {
	return s.correlationID
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
	owner, err := grpc.OwnerFromTokenMD(s.Context(), ownerCache.OwnerClaim())
	if err != nil {
		return nil, grpc.ForwardFromError(codes.InvalidArgument, err)
	}

	devices, err = s.initEventSubscriptions(owner, devices)
	if err != nil {
		_ = s.Close()
		return nil, err
	}
	return devices, nil
}

func (s *Sub) start() error {
	err := s.syncGoroutine.Acquire(s.doneCtx, 1)
	if err != nil {
		return fmt.Errorf("subscription preliminary ends: %w", err)
	}
	go func() {
		defer s.syncGoroutine.Release(1)
		s.run()
	}()
	return nil
}

func (s *Sub) Init(ownerCache *client.OwnerCache) error {
	_, err := s.initOwnerSubscription(ownerCache)
	if err != nil {
		return err
	}
	return s.start()
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

func (s *Sub) initEventSubscriptions(owner string, deviceIDs []string) ([]string, error) {
	var errors []error
	filteredDevices := make([]string, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		if _, ok := s.devicesEventsObserver[deviceID]; ok {
			continue
		}
		obs, err := s.eventsSub.Subscribe(s.Context(), deviceID+"."+s.id, utils.GetDeviceSubject(owner, deviceID), s)
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

func (s *Sub) sendDevicesRegistered(deviceIDs []string) error {
	if !IsFilteredBit(s.filter, FilterBitmaskDeviceRegistered) {
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

func (s *Sub) onRegisteredEvent(e *ownerEvents.DevicesRegistered) {
	devices := s.filterDevices(e.GetDeviceIds())
	devices, err := s.initEventSubscriptions(e.Owner, devices)
	if err != nil {
		s.errFunc(err)
		return
	}
	if len(devices) == 0 {
		return
	}
	err = s.sendDevicesRegistered(devices)
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
	if IsFilteredBit(s.filter, s.filter&FilterBitmaskDeviceUnregistered) {
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
	case <-s.doneCtx.Done():
	case s.ownerCache <- e:
	}
}

func (s *Sub) handleOwnerEvent(e *ownerEvents.Event) {
	switch {
	case e.GetDevicesRegistered() != nil:
		s.onRegisteredEvent(e.GetDevicesRegistered())
	case e.GetDevicesUnregistered() != nil:
		s.onUnregisteredEvent(e.GetDevicesUnregistered())
	}
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
	s.cancelDoneCtx()
	// try to wait for goroutine
	_ = s.syncGoroutine.Acquire(context.Background(), 1)
	// goroutine ended
	s.syncGoroutine.Release(1)
	s.closeOwnerSub()
	return cleanUpDevicesEventsObservers(devicesEventsObserver)
}

// Close closes subscription. Be careful, it will cause a deadlock when you call it from send function.
func (s *Sub) Close() error {
	if s.doneCtx.Err() != nil {
		// is closed
		return nil
	}
	return s.cleanUp(s.devicesEventsObserver)
}

func (s *Sub) run() {
	for {
		select {
		case <-s.doneCtx.Done():
			return
		case event := <-s.eventsCache:
			s.handleEvent(event)
		case ownerEvent := <-s.ownerCache:
			s.handleOwnerEvent(ownerEvent)
		}
	}
}

func New(ctx context.Context, eventsSub *subscriber.Subscriber, send SendEventFunc, correlationID string, cacheSize int, errFunc ErrFunc, req *pb.SubscribeToEvents_CreateSubscription) *Sub {
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
	id := uuid.NewString()
	if errFunc == nil {
		errFunc = func(err error) {}
	} else {
		newErrFunc := func(err error) {
			errFunc(fmt.Errorf("correlationId: %v, subscriptionId: %v: %w", correlationID, id, err))
		}
		errFunc = newErrFunc
	}

	doneCtx, cancelDoneCtx := context.WithCancel(context.Background())

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
		errFunc:               errFunc,
		correlationID:         correlationID,
		devicesEventsObserver: make(map[string]eventbus.Observer),
		eventsCache:           make(chan eventbus.EventUnmarshaler, cacheSize),
		ownerCache:            make(chan *ownerEvents.Event, cacheSize),
		doneCtx:               doneCtx,
		cancelDoneCtx:         cancelDoneCtx,
		ownerSubClose:         ctxSubClose,
		syncGoroutine:         semaphore.NewWeighted(1),
	}
}
