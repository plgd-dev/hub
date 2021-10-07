package subscription

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/pkg/fn"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/kit/v2/strings"
	"go.uber.org/atomic"
)

type SendEventFunc = func(e *pb.Event) error

type Sub struct {
	filter              FilterBitmask
	send                SendEventFunc
	req                 *pb.SubscribeToEvents_CreateSubscription
	id                  string
	correlationID       string
	filteredDeviceIDs   strings.Set
	filteredResourceIDs strings.Set

	closed      atomic.Bool
	closeAtomic atomic.Value
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

func (s *Sub) Id() string {
	return s.id
}

func (s *Sub) CorrelationId() string {
	return s.correlationID
}

func (s *Sub) Init(owner string, subCache *SubscriptionsCache) error {
	subjects := ConvertToSubjects(owner, s.filteredDeviceIDs, s.filteredResourceIDs, s.filter)
	var close fn.FuncList

	for _, subject := range subjects {
		closeSub, err := subCache.Subscribe(subject, s.ProcessEvent)
		if err != nil {
			close.Execute()
			return err
		}
		close.AddFunc(closeSub)
	}

	s.closeAtomic.Store(close.Execute)
	return nil
}

func (s *Sub) isFilteredEvent(e *pb.Event, eventType FilterBitmask) (bool, error) {
	if e == nil {
		return false, fmt.Errorf("invalid event")
	}
	if !IsFilteredBit(s.filter, eventType) {
		return false, nil
	}
	switch ev := e.GetType().(type) {
	case *pb.Event_DeviceRegistered_:
		return true, nil
	case *pb.Event_DeviceUnregistered_:
		return true, nil
	case *pb.Event_ResourcePublished:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourcePublished.GetDeviceId()), nil
	case *pb.Event_ResourceUnpublished:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceUnpublished.GetDeviceId()), nil
	case *pb.Event_ResourceChanged:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceChanged.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, ev.ResourceChanged.AggregateID()), nil
	//case *pb.Event_OperationProcessed_:
	//case *pb.Event_SubscriptionCanceled_:
	case *pb.Event_ResourceUpdatePending:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceUpdatePending.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, ev.ResourceUpdatePending.AggregateID()), nil
	case *pb.Event_ResourceUpdated:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceUpdated.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, ev.ResourceUpdated.AggregateID()), nil
	case *pb.Event_ResourceRetrievePending:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceRetrievePending.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, ev.ResourceRetrievePending.AggregateID()), nil
	case *pb.Event_ResourceRetrieved:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceRetrieved.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, ev.ResourceRetrieved.AggregateID()), nil
	case *pb.Event_ResourceDeletePending:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceDeletePending.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, ev.ResourceDeletePending.AggregateID()), nil
	case *pb.Event_ResourceDeleted:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceDeleted.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, ev.ResourceDeleted.AggregateID()), nil
	case *pb.Event_ResourceCreatePending:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceCreatePending.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, ev.ResourceCreatePending.AggregateID()), nil
	case *pb.Event_ResourceCreated:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceCreated.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, ev.ResourceCreated.AggregateID()), nil
	case *pb.Event_DeviceMetadataUpdatePending:
		return isFilteredDevice(s.filteredDeviceIDs, ev.DeviceMetadataUpdatePending.GroupID()), nil
	case *pb.Event_DeviceMetadataUpdated:
		return isFilteredDevice(s.filteredDeviceIDs, ev.DeviceMetadataUpdated.GroupID()), nil
	}
	return false, fmt.Errorf("unknown event type('%T')", e.GetType())
}

func (s *Sub) ProcessEvent(e *pb.Event, eventType FilterBitmask) error {
	ok, err := s.isFilteredEvent(e, eventType)
	if err != nil {
		return fmt.Errorf("correlationId: %v, subscriptionId: %v: cannot process event ('%v'): %w", s.correlationID, s.Id(), e, err)
	}
	if !ok {
		return nil
	}
	ev := pb.Event{
		SubscriptionId: s.id,
		CorrelationId:  s.correlationID,
		Type:           e.GetType(),
	}
	err = s.send(&ev)
	if err != nil {
		return fmt.Errorf("correlationId: %v, subscriptionId: %v: cannot send event ('%v'): %w", s.correlationID, s.Id(), e, err)
	}
	return nil
}

// Close closes subscription.
func (s *Sub) Close() error {
	if !s.closed.CAS(false, true) {
		return nil
	}
	closeCache := s.closeAtomic.Load().(func())
	closeCache()
	return nil
}

func New(send SendEventFunc, correlationID string, req *pb.SubscribeToEvents_CreateSubscription) *Sub {
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
	var closeAtomic atomic.Value
	closeAtomic.Store(func() {
		// Do nothing because it will be replaced in Init function.
	})
	return &Sub{
		filter:              EventsFilterToBitmask(req.GetEventFilter()),
		send:                send,
		req:                 req,
		id:                  id,
		filteredDeviceIDs:   strings.MakeSet(req.GetDeviceIdFilter()...),
		filteredResourceIDs: filteredResourceIDs,
		correlationID:       correlationID,
		closeAtomic:         closeAtomic,
	}
}
