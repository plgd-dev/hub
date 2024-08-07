package subscription

import (
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/strings"
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

type leadResourceTypeFilter struct {
	enabled bool
	filter  []string
}

func makeLeadResourceTypeFilter(enabled bool, filter []string) leadResourceTypeFilter {
	lrt := leadResourceTypeFilter{
		enabled: enabled,
	}
	if enabled {
		lrt.filter = filter
	}
	return lrt
}

type subjectFilters struct {
	resourceFilters        map[uuid.UUID]*commands.ResourceId
	leadResourceTypeFilter leadResourceTypeFilter
}

type subInit struct {
	filters subjectFilters
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
	for _, filter := range init.filters.resourceFilters {
		wantContinue, err := s.setFilters(filter)
		if err != nil {
			return err
		}
		if !wantContinue {
			break
		}
	}
	subjects := ConvertToSubjects(owner, init.filters, s.filter)
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

//nolint:gocyclo
func (s *Sub) isFilteredEventByType(e *pb.Event) (bool, error) {
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

func (s *Sub) isFilteredEvent(e *pb.Event, eventType FilterBitmask) (bool, error) {
	if e == nil {
		return false, errors.New("invalid event")
	}
	if !IsFilteredBit(s.filter, eventType) {
		return false, nil
	}
	return s.isFilteredEventByType(e)
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

func toFilters(req *pb.SubscribeToEvents_CreateSubscription) ([]string, []string, []string) {
	filterDeviceIDs := make([]string, 0, len(req.GetDeviceIdFilter())+len(req.GetResourceIdFilter()))
	filterHrefs := make([]string, 0, len(req.GetHrefFilter())+len(req.GetResourceIdFilter()))
	filterResourceIDs := make([]string, 0, len(req.GetResourceIdFilter()))
	for _, r := range req.GetResourceIdFilter() {
		v := r.GetResourceId()
		if v != nil {
			switch {
			case v.GetDeviceId() == "*" && v.GetHref() == "*":
				continue
			case v.GetDeviceId() == "*":
				filterHrefs = append(filterHrefs, v.GetHref())
			case v.GetHref() == "*":
				filterDeviceIDs = append(filterDeviceIDs, v.GetDeviceId())
			default:
				filterResourceIDs = append(filterResourceIDs, v.ToString())
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

	return strings.Unique(filterDeviceIDs), strings.Unique(filterHrefs), strings.Unique(filterResourceIDs), bitmask
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

func addResourceIdFilters(filterSlice []*pb.ResourceIdFilter, deviceHrefFilters map[uuid.UUID]*commands.ResourceId) {
	for _, r := range filterSlice {
		v := r.GetResourceId()
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

func getLeadResourceTypeFilter(req *pb.SubscribeToEvents_CreateSubscription, leadRTFilterEnabled bool) leadResourceTypeFilter {
	if !leadRTFilterEnabled {
		return makeLeadResourceTypeFilter(false, nil)
	}

	var lrtFilter []string
	if slices.Contains(req.GetLeadResourceTypeFilter(), ">") {
		// - in NATS the ">" wildcard matches one or more subjects, so there is no need for any other filters when leadRTFilter is ">"
		// - "*" will subscribe to all events that have been published with lead resource type, but not events that have been published without it
		// - if no leadRTFilter is provided, then it will subscribe events with and without a lead resource type
		lrtFilter = []string{">"}
	} else {
		lrtFilter = strings.Unique(req.GetLeadResourceTypeFilter())
	}
	return makeLeadResourceTypeFilter(true, lrtFilter)
}

func getFilters(req *pb.SubscribeToEvents_CreateSubscription, leadRTFilterEnabled bool) (subjectFilters, FilterBitmask) {
	filterDeviceIDs, filterHrefs, filterResourceIDs, bitmask := normalizeFilters(req)

	leadRTFilter := getLeadResourceTypeFilter(req, leadRTFilterEnabled)
	// all events or just filtered by leading resource type
	if len(filterDeviceIDs) == 0 && len(filterHrefs) == 0 && len(filterResourceIDs) == 0 {
		return subjectFilters{
			leadResourceTypeFilter: leadRTFilter,
		}, bitmask
	}

	sf := subjectFilters{
		resourceFilters:        make(map[uuid.UUID]*commands.ResourceId),
		leadResourceTypeFilter: leadRTFilter,
	}

	if len(req.GetResourceIdFilter()) == 0 {
		if len(filterDeviceIDs) == 0 {
			addHrefFilters(filterHrefs, sf.resourceFilters)
			return sf, bitmask
		}

		if len(filterHrefs) == 0 {
			addDeviceFilters(filterDeviceIDs, sf.resourceFilters)
			return sf, bitmask
		}
	}

	if len(filterDeviceIDs) == 0 {
		if len(filterHrefs) != 0 {
			addHrefFilters(filterHrefs, sf.resourceFilters)
		}
		addResourceIdFilters(req.GetResourceIdFilter(), sf.resourceFilters)
		return sf, bitmask
	}

	if len(filterHrefs) == 0 {
		addDeviceFilters(filterDeviceIDs, sf.resourceFilters)
		addResourceIdFilters(req.GetResourceIdFilter(), sf.resourceFilters)
		return sf, bitmask
	}

	for _, d := range filterDeviceIDs {
		for _, h := range filterHrefs {
			sf.resourceFilters[commands.NewResourceID(d, h).ToUUID()] = commands.NewResourceID(d, h)
		}
	}

	addResourceIdFilters(req.GetResourceIdFilter(), sf.resourceFilters)

	return sf, bitmask
}

func New(send SendEventFunc, correlationID string, leadRTEnabled bool, req *pb.SubscribeToEvents_CreateSubscription) *Sub {
	// for backward compatibility and http api
	req.ResourceIdFilter = append(req.ResourceIdFilter, req.ConvertHTTPResourceIDFilter()...)

	filters, bitmask := getFilters(req, leadRTEnabled)
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
			filters: filters,
		},
		filteredHrefIDs:     make(set),
		filteredDeviceIDs:   make(set),
		filteredResourceIDs: make(set),
		correlationID:       correlationID,
		closeAtomic:         closeAtomic,
	}
}
