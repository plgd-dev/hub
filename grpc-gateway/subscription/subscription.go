package subscription

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"go.uber.org/atomic"
)

type SendEventFunc = func(e *pb.Event) error

type set map[uuid.UUID]struct{}

func (s set) Has(a uuid.UUID) bool {
	if len(a) == 0 {
		return true
	}
	_, ok := s[a]
	return ok
}

type subInit struct {
	deviceHrefFilters map[uuid.UUID]*commands.ResourceId
}

type Sub struct {
	filter        FilterBitmask
	send          SendEventFunc
	req           *pb.SubscribeToEvents_CreateSubscription
	id            string
	correlationID string
	init          *subInit

	filteredDeviceIDs   set
	filteredHrefIDs     set
	filteredResourceIDs set

	closed      atomic.Bool
	closeAtomic atomic.Value
}

func isFilteredDevice(filteredDeviceIDs set, deviceID string) bool {
	if len(filteredDeviceIDs) == 0 {
		return true
	}
	if v, err := uuid.Parse(deviceID); err == nil {
		return filteredDeviceIDs.Has(v)
	}
	return false
}

func isFilteredResourceIDs(filteredResourceIDs set, filteredHrefs set, resourceID *commands.ResourceId) bool {
	if len(filteredHrefs) == 0 && len(filteredResourceIDs) == 0 {
		return true
	}
	if resourceID == nil {
		return false
	}
	return filteredHrefs.Has(utils.HrefToID(resourceID.GetHref())) || filteredResourceIDs.Has(resourceID.ToUUID())
}

func (s *Sub) Id() string {
	return s.id
}

func (s *Sub) CorrelationId() string {
	return s.correlationID
}

func (s *Sub) setFilters(filter *commands.ResourceId) (bool, error) {
	if filter.GetDeviceId() == "*" && filter.GetHref() == "*" {
		s.filteredHrefIDs = make(set)
		s.filteredDeviceIDs = make(set)
		s.filteredResourceIDs = make(set)
		return false, nil
	}
	if filter.GetDeviceId() != "*" && filter.GetHref() != "*" {
		d, err := uuid.Parse(filter.GetDeviceId())
		if err != nil {
			return false, fmt.Errorf("invalid deviceId('%v'): %w", filter.GetDeviceId(), err)
		}
		s.filteredResourceIDs[filter.ToUUID()] = struct{}{}
		s.filteredDeviceIDs[d] = struct{}{}
		return true, nil
	}
	if filter.GetHref() != "*" {
		h := utils.HrefToID(filter.GetHref())
		s.filteredHrefIDs[h] = struct{}{}
	}
	if filter.GetDeviceId() != "*" {
		d, err := uuid.Parse(filter.GetDeviceId())
		if err != nil {
			return false, fmt.Errorf("invalid deviceId('%v'): %w", filter.GetDeviceId(), err)
		}
		s.filteredDeviceIDs[d] = struct{}{}
	}
	return true, nil
}

func (s *Sub) Init(owner string, subCache *SubscriptionsCache) error {
	init := s.init
	s.init = nil
	for _, filter := range init.deviceHrefFilters {
		wantContinue, err := s.setFilters(filter)
		if err != nil {
			return err
		}
		if !wantContinue {
			break
		}
	}
	subjects := ConvertToSubjects(owner, init.deviceHrefFilters, s.filter)
	var closeFn fn.FuncList
	for _, subject := range subjects {
		closeSub, err := subCache.Subscribe(subject, s.ProcessEvent)
		if err != nil {
			closeFn.Execute()
			return err
		}
		closeFn.AddFunc(closeSub)
	}

	s.closeAtomic.Store(closeFn.Execute)
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
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceChanged.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, s.filteredHrefIDs, ev.ResourceChanged.GetResourceId()), nil
	// case *pb.Event_OperationProcessed_:
	// case *pb.Event_SubscriptionCanceled_:
	case *pb.Event_ResourceUpdatePending:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceUpdatePending.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, s.filteredHrefIDs, ev.ResourceUpdatePending.GetResourceId()), nil
	case *pb.Event_ResourceUpdated:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceUpdated.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, s.filteredHrefIDs, ev.ResourceUpdated.GetResourceId()), nil
	case *pb.Event_ResourceRetrievePending:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceRetrievePending.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, s.filteredHrefIDs, ev.ResourceRetrievePending.GetResourceId()), nil
	case *pb.Event_ResourceRetrieved:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceRetrieved.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, s.filteredHrefIDs, ev.ResourceRetrieved.GetResourceId()), nil
	case *pb.Event_ResourceDeletePending:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceDeletePending.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, s.filteredHrefIDs, ev.ResourceDeletePending.GetResourceId()), nil
	case *pb.Event_ResourceDeleted:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceDeleted.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, s.filteredHrefIDs, ev.ResourceDeleted.GetResourceId()), nil
	case *pb.Event_ResourceCreatePending:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceCreatePending.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, s.filteredHrefIDs, ev.ResourceCreatePending.GetResourceId()), nil
	case *pb.Event_ResourceCreated:
		return isFilteredDevice(s.filteredDeviceIDs, ev.ResourceCreated.GroupID()) && isFilteredResourceIDs(s.filteredResourceIDs, s.filteredHrefIDs, ev.ResourceCreated.GetResourceId()), nil
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
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	closeCache := s.closeAtomic.Load().(func())
	closeCache()
	return nil
}

func removeDuplicateStrings(s []string) []string {
	if len(s) < 1 {
		return s
	}

	sort.Strings(s)
	prev := 1
	for curr := 1; curr < len(s); curr++ {
		if s[curr-1] != s[curr] {
			s[prev] = s[curr]
			prev++
		}
	}

	return s[:prev]
}

func toFilters(req *pb.SubscribeToEvents_CreateSubscription) ([]string, []string, []string) {
	filterDeviceIDs := make([]string, 0, len(req.GetDeviceIdFilter())+len(req.GetResourceIdFilter()))
	filterHrefs := make([]string, 0, len(req.GetHrefFilter())+len(req.GetResourceIdFilter()))
	filterResourceIDs := make([]string, 0, len(req.GetResourceIdFilter()))
	for _, r := range req.GetResourceIdFilter() {
		v := commands.ResourceIdFromString(r)
		if v != nil {
			switch {
			case v.GetDeviceId() == "*" && v.GetHref() == "*":
				continue
			case v.GetDeviceId() == "*":
				filterHrefs = append(filterHrefs, v.GetHref())
			case v.GetHref() == "*":
				filterDeviceIDs = append(filterDeviceIDs, v.GetDeviceId())
			default:
				filterResourceIDs = append(filterResourceIDs, r)
			}
		}
	}
	return filterDeviceIDs, filterHrefs, filterResourceIDs
}

func normalizeFilters(req *pb.SubscribeToEvents_CreateSubscription) ([]string, []string, []string, FilterBitmask) {
	bitmask := EventsFilterToBitmask(req.GetEventFilter())
	filterDeviceIDs, filterHrefs, filterResourceIDs := toFilters(req)

	for _, d := range req.GetDeviceIdFilter() {
		if d != "*" {
			filterDeviceIDs = append(filterDeviceIDs, d)
		}
	}
	for _, h := range req.GetHrefFilter() {
		if h != "*" {
			filterHrefs = append(filterHrefs, h)
		}
	}
	if len(filterResourceIDs) == 0 {
		filterResourceIDs = nil
	}
	if len(filterDeviceIDs) == 0 {
		filterDeviceIDs = nil
	}
	if len(filterHrefs) == 0 {
		filterHrefs = nil
	}

	return removeDuplicateStrings(filterDeviceIDs), removeDuplicateStrings(filterHrefs), removeDuplicateStrings(filterResourceIDs), bitmask
}

func addHrefFilters(filterSlice []string, deviceHrefFilters map[uuid.UUID]*commands.ResourceId) {
	for _, h := range filterSlice {
		deviceHrefFilters[utils.HrefToID(h)] = commands.NewResourceID("*", h)
	}
}

func addDeviceFilters(filterSlice []string, deviceHrefFilters map[uuid.UUID]*commands.ResourceId) {
	for _, d := range filterSlice {
		deviceHrefFilters[utils.HrefToID(d+"/*")] = commands.NewResourceID(d, "*")
	}
}

func addResourceIdFilters(filterSlice []string, deviceHrefFilters map[uuid.UUID]*commands.ResourceId) {
	for _, r := range filterSlice {
		v := commands.ResourceIdFromString(r)
		if v == nil {
			// invalid resource id - skip
			continue
		}
		if _, ok := deviceHrefFilters[utils.HrefToID(v.GetHref())]; ok {
			// already added - skip because it is more specific than '*/href' subscription
			continue
		}
		if _, ok := deviceHrefFilters[utils.HrefToID(v.GetDeviceId()+"/*")]; ok {
			// already added - skip because it is more specific than 'deviceId/*' subscription
			continue
		}
		deviceHrefFilters[v.ToUUID()] = v
	}
}

func getFilters(req *pb.SubscribeToEvents_CreateSubscription) (map[uuid.UUID]*commands.ResourceId, FilterBitmask) {
	filterDeviceIDs, filterHrefs, filterResourceIDs, bitmask := normalizeFilters(req)
	// all events
	if len(filterDeviceIDs) == 0 && len(filterHrefs) == 0 && len(filterResourceIDs) == 0 {
		return nil, bitmask
	}
	deviceHrefFilters := make(map[uuid.UUID]*commands.ResourceId)

	if len(filterDeviceIDs) == 0 && len(req.GetResourceIdFilter()) == 0 {
		addHrefFilters(filterHrefs, deviceHrefFilters)
		return deviceHrefFilters, bitmask
	}

	if len(filterHrefs) == 0 && len(req.GetResourceIdFilter()) == 0 {
		addDeviceFilters(filterDeviceIDs, deviceHrefFilters)
		return deviceHrefFilters, bitmask
	}

	if len(filterDeviceIDs) == 0 && len(filterHrefs) == 0 {
		addResourceIdFilters(req.GetResourceIdFilter(), deviceHrefFilters)
		return deviceHrefFilters, bitmask
	}

	if len(filterDeviceIDs) == 0 {
		addHrefFilters(filterHrefs, deviceHrefFilters)
		addResourceIdFilters(req.GetResourceIdFilter(), deviceHrefFilters)
		return deviceHrefFilters, bitmask
	}

	if len(filterHrefs) == 0 {
		addDeviceFilters(filterDeviceIDs, deviceHrefFilters)
		addResourceIdFilters(req.GetResourceIdFilter(), deviceHrefFilters)
		return deviceHrefFilters, bitmask
	}

	for _, d := range filterDeviceIDs {
		for _, h := range filterHrefs {
			deviceHrefFilters[commands.NewResourceID(d, h).ToUUID()] = commands.NewResourceID(d, h)
		}
	}

	addResourceIdFilters(req.GetResourceIdFilter(), deviceHrefFilters)

	return deviceHrefFilters, bitmask
}

func New(send SendEventFunc, correlationID string, req *pb.SubscribeToEvents_CreateSubscription) *Sub {
	deviceHrefFilters, bitmask := getFilters(req)
	id := uuid.NewString()
	var closeAtomic atomic.Value
	closeAtomic.Store(func() {
		// Do nothing because it will be replaced in Init function.
	})
	return &Sub{
		filter: bitmask,
		send:   send,
		req:    req,
		id:     id,
		init: &subInit{
			deviceHrefFilters: deviceHrefFilters,
		},
		filteredHrefIDs:     make(set),
		filteredDeviceIDs:   make(set),
		filteredResourceIDs: make(set),
		correlationID:       correlationID,
		closeAtomic:         closeAtomic,
	}
}
