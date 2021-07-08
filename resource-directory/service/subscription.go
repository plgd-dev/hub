package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/kit/strings"
	kitSync "github.com/plgd-dev/kit/sync"
)

type subscriptionEvent struct {
	version uint64
}

type subscription struct {
	devicesEvent *pb.SubscribeToEvents_CreateSubscription

	isInitialized       sync.Map
	filteredDeviceIDs   strings.Set
	filteredResourceIDs strings.Set
	filteredEvents      filterBitmask

	id            string
	userID        string
	correlationID string

	resourceProjection *Projection
	send               SendEventFunc

	lock                          sync.Mutex
	registeredDevicesInProjection map[string]bool

	eventVersionsLock sync.Mutex
	eventVersions     map[string]subscriptionEvent

	isInitializedResource *kitSync.Map
}

func Newsubscription(id, userID, correlationID string, send SendEventFunc, resourceProjection *Projection, devicesEvent *pb.SubscribeToEvents_CreateSubscription) *subscription {
	filteredDeviceIDs := strings.MakeSet(devicesEvent.GetDeviceIdFilter()...)
	filteredResourceIDs := strings.MakeSet()
	if len(devicesEvent.GetResourceIdFilter()) > 0 {
		filteredDeviceIDs = strings.MakeSet()
	}
	for _, r := range devicesEvent.GetResourceIdFilter() {
		res := commands.ResourceIdFromString(r)
		filteredResourceIDs.Add(res.ToUUID())
		filteredDeviceIDs.Add(res.GetDeviceId())
	}
	return &subscription{
		userID:                        userID,
		id:                            id,
		correlationID:                 correlationID,
		send:                          send,
		resourceProjection:            resourceProjection,
		eventVersions:                 make(map[string]subscriptionEvent),
		registeredDevicesInProjection: make(map[string]bool),
		devicesEvent:                  devicesEvent,
		filteredDeviceIDs:             filteredDeviceIDs,
		filteredResourceIDs:           filteredResourceIDs,
		isInitializedResource:         kitSync.NewMap(),
		filteredEvents:                devicesEventsFilterToBitmask(devicesEvent.GetEventFilter()),
	}
}

func (s *subscription) update(ctx context.Context, currentDevices map[string]bool, init bool) error {
	filteredDevices := make([]string, 0, 32)
	for deviceID := range currentDevices {
		if isFilteredDevice(s.filteredDeviceIDs, deviceID) {
			_, err := s.RegisterToProjection(ctx, deviceID)
			if err != nil {
				log.Errorf("cannot register to resource projection for %v: %v", deviceID, err)
				continue
			}
			filteredDevices = append(filteredDevices, deviceID)
		}

	}

	if init || len(filteredDevices) > 0 {
		err := s.NotifyOfRegisteredDevice(ctx, filteredDevices)
		if err != nil {
			return err
		}
	}

	err := s.initDevices(ctx, filteredDevices)
	if err != nil {
		return err
	}
	return nil
}

func (s *subscription) Init(ctx context.Context, currentDevices map[string]bool) error {
	return s.update(ctx, currentDevices, true)
}

func (s *subscription) Update(ctx context.Context, addedDevices, removedDevices map[string]bool) error {
	toSend := make([]string, 0, 32)
	for deviceID := range removedDevices {
		devID := deviceID
		toSend = append(toSend, devID)
		err := s.UnregisterFromProjection(ctx, deviceID)
		if err != nil {
			log.Errorf("cannot unregister resource from projection for %v: %v", deviceID, err)
		}
	}
	if len(toSend) > 0 {
		err := s.NotifyOfUnregisteredDevice(ctx, toSend)
		if err != nil {
			return fmt.Errorf("cannot send device unregistered: %w", err)
		}
	}
	return s.update(ctx, addedDevices, false)
}

func (s *subscription) initDevices(ctx context.Context, deviceIDs []string) error {
	var errors []error
	initFuncs := []func(ctx context.Context, deviceID string) error{
		s.initNotifyOfDevicesMetadata,
		s.initSendResourcesPublished,
		s.initSendResourcesChanged,
		s.initSendResourcesCreatePending,
		s.initSendResourcesDeletePending,
		s.initSendResourcesUpdatePending,
		s.initSendResourcesRetrievePending,
	}
	for _, deviceID := range deviceIDs {
		for _, f := range initFuncs {
			err := f(ctx, deviceID)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

func isFilteredDevice(filteredDeviceIDs strings.Set, deviceID string) bool {
	if len(filteredDeviceIDs) == 0 {
		return true
	}
	return filteredDeviceIDs.HasOneOf(deviceID)
}

func isFilteredResourceIDs(filteredResourceIDs strings.Set, resourceID *commands.ResourceId) bool {
	if len(filteredResourceIDs) == 0 {
		return true
	}
	return filteredResourceIDs.HasOneOf(resourceID.ToUUID())
}

func (s *subscription) Filter(resourceID *commands.ResourceId, typeEvent string, version uint64) bool {
	if _, ok := s.isInitialized.Load(resourceID.GetDeviceId()); !ok {
		return false
	}
	if !isFilteredDevice(s.filteredDeviceIDs, resourceID.GetDeviceId()) {
		return false
	}
	if !isFilteredResourceIDs(s.filteredResourceIDs, resourceID) {
		return false
	}
	return s.filterByVersion(resourceID, typeEvent, version)
}

type isInitialized struct {
	sync.Mutex
	initialized bool
}

func (s *subscription) initializeResource(resourceID *commands.ResourceId, isInit bool) bool {
	if isInit {
		value, _ := s.isInitializedResource.LoadOrStoreWithFunc(resourceID.ToUUID(), func(value interface{}) interface{} {
			v := value.(*isInitialized)
			v.Lock()
			return v
		}, func() interface{} {
			var v isInitialized
			v.Lock()
			return &v
		})
		v := value.(*isInitialized)
		v.initialized = true
		defer v.Unlock()
		return v.initialized
	}
	value, ok := s.isInitializedResource.LoadWithFunc(resourceID.ToUUID(), func(value interface{}) interface{} {
		v := value.(*isInitialized)
		v.Lock()
		return v
	})
	if ok {
		v := value.(*isInitialized)
		defer v.Unlock()
		return v.initialized
	}
	return false
}

func (s *subscription) UserID() string {
	return s.userID
}

func (s *subscription) ID() string {
	return s.id
}

func (s *subscription) CorrelationID() string {
	return s.correlationID
}

func (s *subscription) filterByVersion(resourceID *commands.ResourceId, typeEvent string, version uint64) bool {
	rID := resourceID.ToUUID()
	s.eventVersionsLock.Lock()
	defer s.eventVersionsLock.Unlock()
	v, ok := s.eventVersions[rID]
	if !ok {
		s.eventVersions[rID] = subscriptionEvent{
			version: version,
		}
		return true
	}
	if v.version >= version {
		return false
	}
	v.version = version
	s.eventVersions[rID] = v
	return true
}

func (s *subscription) RegisterToProjection(ctx context.Context, deviceID string) (loaded bool, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	loaded, err = s.resourceProjection.Register(ctx, deviceID)
	if err != nil {
		return loaded, err
	}
	s.registeredDevicesInProjection[deviceID] = true
	return
}

func (s *subscription) UnregisterFromProjection(ctx context.Context, deviceID string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, ok := s.registeredDevicesInProjection[deviceID]
	if !ok {
		return nil
	}
	delete(s.registeredDevicesInProjection, deviceID)
	return s.resourceProjection.Unregister(deviceID)
}

func (s *subscription) Send(event *pb.Event) error {
	return s.send(event)
}

func (s *subscription) Close(reason error) error {
	r := ""
	if reason != nil {
		r = reason.Error()
	}

	var errors []error

	err := s.unregisterProjections()
	if err != nil {
		errors = append(errors, err)
	}

	err = s.Send(&pb.Event{
		CorrelationId:  s.CorrelationID(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_SubscriptionCanceled_{
			SubscriptionCanceled: &pb.Event_SubscriptionCanceled{
				Reason: r,
			},
		},
	})
	if err != nil {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot close subscription %v: %v", s.ID(), errors)
	}

	return nil
}

func (s *subscription) unregisterProjections() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	var errors []error
	for deviceID := range s.registeredDevicesInProjection {
		err := s.resourceProjection.Unregister(deviceID)
		if err != nil {
			errors = append(errors, err)
		}
		delete(s.registeredDevicesInProjection, deviceID)
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot unregister projections for %v: %v", s.ID(), errors)
	}
	return nil
}
