package client

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	grpcSubscription "github.com/plgd-dev/hub/grpc-gateway/subscription"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/kit/v2/strings"
	"go.uber.org/atomic"
)

type SendEventFunc = func(e *pb.Event) error

type deduplicateEvent struct {
	version    uint64
	validUntil *time.Time
}

type Sub struct {
	ctx                 atomic.Value
	filter              grpcSubscription.FilterBitmask
	send                SendEventFunc
	req                 *pb.SubscribeToEvents_CreateSubscription
	correlationID       string
	id                  string
	expiration          time.Duration
	devicesInitialized  map[string]bool
	filteredDeviceIDs   strings.Set
	filteredResourceIDs strings.Set
	grpcClient          pb.GrpcGatewayClient
	deduplicateEvents   map[string]deduplicateEvent
}

func isFilteredDevice(filteredDeviceIDs strings.Set, deviceID string) bool {
	if len(filteredDeviceIDs) == 0 {
		return true
	}
	return filteredDeviceIDs.HasOneOf(deviceID)
}

func (s *Sub) deinitDeviceLocked(deviceID string) {
	delete(s.devicesInitialized, deviceID)
}

func (s *Sub) SetContext(ctx context.Context) {
	s.ctx.Store(ctx)
}

func (s *Sub) Context() context.Context {
	return s.ctx.Load().(context.Context)
}

func (s *Sub) initDevices() ([]string, error) {
	devicesClient, err := s.grpcClient.GetDevices(s.Context(), &pb.GetDevicesRequest{
		DeviceIdFilter: s.req.GetDeviceIdFilter(),
	})
	errFunc := func(err error) error {
		return fmt.Errorf("cannot init devices events for '%v': %w", s.req.GetDeviceIdFilter(), err)
	}
	if err != nil {
		return nil, errFunc(fmt.Errorf("cannot get devices: %w", err))
	}
	devices := make([]string, 0, 32)
	for {
		recv, err := devicesClient.Recv()
		if err == io.EOF {
			return devices, nil
		}
		if err != nil {
			return nil, errFunc(fmt.Errorf("cannot receive resource: %w", err))
		}
		devices = append(devices, recv.GetId())
	}
}

func (s *Sub) initSubscription() ([]string, error) {
	devices, err := s.initDevices()
	if err != nil {
		return nil, err
	}

	devices = s.initEventSubscriptions(devices)
	return devices, nil
}

func (s *Sub) initEvents(devices []string) error {
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

func (s *Sub) Init(id string) error {
	s.id = id
	devices, err := s.initSubscription()
	if err != nil {
		return err
	}
	err = s.initEvents(devices)
	if err != nil {
		return err
	}
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

func (s *Sub) initEventSubscriptions(deviceIDs []string) []string {
	filteredDevices := make([]string, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		if _, ok := s.devicesInitialized[deviceID]; ok {
			continue
		}
		s.devicesInitialized[deviceID] = true
		filteredDevices = append(filteredDevices, deviceID)
	}
	return filteredDevices
}

func (s *Sub) sendDevicesRegistered(deviceIDs []string, validUntil *time.Time) error {
	if !grpcSubscription.IsFilteredBit(s.filter, grpcSubscription.FilterBitmaskDeviceRegistered) {
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
	if !grpcSubscription.IsFilteredBit(s.filter, grpcSubscription.FilterBitmaskResourceChanged) {
		return nil
	}
	errFunc := func(err error) error {
		return fmt.Errorf("cannot init resources changed events for devices %v: %w", deviceIDs, err)
	}
	deviceIdFilter := deviceIDs
	if len(s.req.GetResourceIdFilter()) > 0 {
		deviceIdFilter = nil
	}
	resourcesClient, err := s.grpcClient.GetResources(s.Context(), &pb.GetResourcesRequest{
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
	if !grpcSubscription.IsFilteredBit(s.filter, grpcSubscription.FilterBitmaskDeviceMetadataUpdated) {
		return nil
	}
	errFunc := func(err error) error {
		return fmt.Errorf("cannot init devices metadata for devices %v: %w", deviceIDs, err)
	}
	linksClient, err := s.grpcClient.GetDevicesMetadata(s.Context(), &pb.GetDevicesMetadataRequest{
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
	if !grpcSubscription.IsFilteredBit(s.filter, grpcSubscription.FilterBitmaskResourcesPublished) {
		return nil
	}
	errFunc := func(err error) error {
		return fmt.Errorf("cannot init resources published events for devices %v: %w", deviceIDs, err)
	}
	linksClient, err := s.grpcClient.GetResourceLinks(s.Context(), &pb.GetResourceLinksRequest{
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
	if !grpcSubscription.IsFilteredBit(s.filter,
		grpcSubscription.FilterBitmaskDeviceMetadataUpdatePending|
			grpcSubscription.FilterBitmaskResourceCreatePending|
			grpcSubscription.FilterBitmaskResourceRetrievePending|
			grpcSubscription.FilterBitmaskResourceUpdatePending|
			grpcSubscription.FilterBitmaskResourceDeletePending) {
		return nil
	}
	errFunc := func(err error) error {
		return fmt.Errorf("cannot init pending commands for devices %v: %w", deviceIDs, err)
	}

	deviceIdFilter := deviceIDs
	if len(s.req.GetResourceIdFilter()) > 0 {
		deviceIdFilter = nil
	}

	pendingCommands, err := s.grpcClient.GetPendingCommands(s.Context(), &pb.GetPendingCommandsRequest{
		DeviceIdFilter:   deviceIdFilter,
		ResourceIdFilter: s.req.GetResourceIdFilter(),
		CommandFilter:    grpcSubscription.BitmaskToFilterPendingsCommands(s.filter),
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

func (s *Sub) onRegisteredEvent(e *pb.Event_DeviceRegistered) error {
	devices := s.filterDevices(e.GetDeviceIds())
	devices = s.initEventSubscriptions(devices)
	if len(devices) == 0 {
		return nil
	}
	return s.initEvents(devices)
}

func (s *Sub) onUnregisteredEvent(e *pb.Event_DeviceUnregistered) error {
	devices := s.filterDevices(e.GetDeviceIds())
	if len(devices) == 0 {
		return nil
	}
	for _, deviceID := range devices {
		s.deinitDeviceLocked(deviceID)
	}
	if !grpcSubscription.IsFilteredBit(s.filter, s.filter&grpcSubscription.FilterBitmaskDeviceUnregistered) {
		return nil
	}
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
		return fmt.Errorf("cannot send device unregistered event for devices %v: %w", devices, err)
	}
	return nil
}

func (s *Sub) dropExpiredDeduplicateEvents(now time.Time) {
	for key, val := range s.deduplicateEvents {
		if val.validUntil == nil || now.After(*val.validUntil) {
			delete(s.deduplicateEvents, key)
		}
	}
}

func (s *Sub) DropDeduplicateEvents() {
	s.deduplicateEvents = make(map[string]deduplicateEvent)
}

func (s *Sub) ProcessEvent(e *pb.Event) error {
	s.dropExpiredDeduplicateEvents(time.Now())
	var evToCheck event
	switch ev := e.GetType().(type) {
	case (*pb.Event_DeviceRegistered_):
		return s.onRegisteredEvent(ev.DeviceRegistered)
	case (*pb.Event_DeviceUnregistered_):
		return s.onUnregisteredEvent(ev.DeviceUnregistered)
	case (*pb.Event_ResourcePublished):
		evToCheck = ev.ResourcePublished
	case (*pb.Event_ResourceChanged):
		evToCheck = ev.ResourceChanged
	case (*pb.Event_ResourceUpdatePending):
		evToCheck = ev.ResourceUpdatePending
	case (*pb.Event_ResourceRetrievePending):
		evToCheck = ev.ResourceRetrievePending
	case (*pb.Event_ResourceDeletePending):
		evToCheck = ev.ResourceDeletePending
	case (*pb.Event_ResourceCreatePending):
		evToCheck = ev.ResourceCreatePending
	case (*pb.Event_DeviceMetadataUpdatePending):
		evToCheck = ev.DeviceMetadataUpdatePending
	default:
		return s.send(e)
	}

	if evToCheck != nil && s.isDuplicatedEvent(evToCheck) {
		return nil
	}
	return s.send(e)
}

func NewSub(ctx context.Context, grpcClient pb.GrpcGatewayClient, send SendEventFunc, correlationID string, expiration time.Duration, req *pb.SubscribeToEvents_CreateSubscription) *Sub {
	bitmask := grpcSubscription.EventsFilterToBitmask(req.GetEventFilter())
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
			if bitmask&(grpcSubscription.FilterBitmaskDeviceMetadataUpdatePending|grpcSubscription.FilterBitmaskDeviceMetadataUpdated) != 0 {
				filteredResourceIDs.Add(commands.MakeStatusResourceUUID(v.GetDeviceId()))
			}
			if bitmask&(grpcSubscription.FilterBitmaskResourcesPublished|grpcSubscription.FilterBitmaskResourcesUnpublished) != 0 {
				filteredResourceIDs.Add(commands.MakeLinksResourceUUID(v.GetDeviceId()))
			}
		}
	}
	if expiration <= 0 {
		expiration = time.Second * 60
	}
	var ctxAtomic atomic.Value
	ctxAtomic.Store(ctx)

	return &Sub{
		ctx:                 ctxAtomic,
		filter:              grpcSubscription.EventsFilterToBitmask(req.GetEventFilter()),
		send:                send,
		req:                 req,
		filteredDeviceIDs:   strings.MakeSet(req.GetDeviceIdFilter()...),
		filteredResourceIDs: filteredResourceIDs,
		grpcClient:          grpcClient,
		correlationID:       correlationID,
		expiration:          expiration,
		deduplicateEvents:   make(map[string]deduplicateEvent),
		devicesInitialized:  make(map[string]bool),
	}
}
