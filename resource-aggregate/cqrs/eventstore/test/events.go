package test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	pkgTime "github.com/plgd-dev/hub/pkg/time"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/resource-aggregate/events"
)

var StaticAuditContext = &commands.AuditContext{
	UserId: "userId",
}

func MakeResourceLinksPublishedEvent(resources []*commands.Resource, deviceID string, eventMetadata *events.EventMetadata) eventstore.EventUnmarshaler {
	e := events.ResourceLinksPublished{
		Resources:     resources,
		DeviceId:      deviceID,
		AuditContext:  StaticAuditContext,
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceLinksPublished{}).EventType(),
		commands.MakeLinksResourceUUID(e.GetDeviceId()),
		e.GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceLinksPublished); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceLinksUnpublishedEvent(hrefs []string, deviceID string, eventMetadata *events.EventMetadata) eventstore.EventUnmarshaler {
	e := events.ResourceLinksUnpublished{
		Hrefs:         hrefs,
		DeviceId:      deviceID,
		AuditContext:  StaticAuditContext,
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceLinksUnpublished{}).EventType(),
		commands.MakeLinksResourceUUID(e.GetDeviceId()),
		e.GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceLinksUnpublished); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceLinksSnapshotTaken(resources map[string]*commands.Resource, deviceID string, eventMetadata *events.EventMetadata) eventstore.EventUnmarshaler {
	e := events.NewResourceLinksSnapshotTaken()
	e.Resources = resources
	e.DeviceId = deviceID
	e.EventMetadata = eventMetadata

	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceLinksSnapshotTaken{}).EventType(),
		commands.MakeLinksResourceUUID(e.GetDeviceId()),
		e.GetDeviceId(),
		true,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceLinksSnapshotTaken); ok {
				x.CopyData(e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeAuditContext(userID string, correlationID string) *commands.AuditContext {
	return &commands.AuditContext{
		UserId:        userID,
		CorrelationId: correlationID,
	}
}

func MakeResourceUpdatePending(resourceId *commands.ResourceId, content *commands.Content, eventMetadata *events.EventMetadata, auditContext *commands.AuditContext, validUntil time.Time) eventstore.EventUnmarshaler {
	e := events.ResourceUpdatePending{
		ResourceId:    resourceId,
		Content:       content,
		AuditContext:  auditContext,
		EventMetadata: eventMetadata,
		ValidUntil:    pkgTime.UnixNano(validUntil),
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceUpdatePending{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceUpdatePending); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceUpdated(resourceId *commands.ResourceId, status commands.Status, content *commands.Content, eventMetadata *events.EventMetadata, auditContext *commands.AuditContext) eventstore.EventUnmarshaler {
	e := events.ResourceUpdated{
		ResourceId:    resourceId,
		Content:       content,
		Status:        status,
		AuditContext:  auditContext,
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceUpdated{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceUpdated); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceCreatePending(resourceId *commands.ResourceId, content *commands.Content, eventMetadata *events.EventMetadata, auditContext *commands.AuditContext, validUntil time.Time) eventstore.EventUnmarshaler {
	e := events.ResourceCreatePending{
		ResourceId:    resourceId,
		Content:       content,
		AuditContext:  auditContext,
		EventMetadata: eventMetadata,
		ValidUntil:    pkgTime.UnixNano(validUntil),
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceCreatePending{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceCreatePending); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceCreated(resourceId *commands.ResourceId, status commands.Status, content *commands.Content, eventMetadata *events.EventMetadata, auditContext *commands.AuditContext) eventstore.EventUnmarshaler {
	e := events.ResourceCreated{
		ResourceId:    resourceId,
		Content:       content,
		Status:        status,
		AuditContext:  auditContext,
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceCreated{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceCreated); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceChangedEvent(resourceId *commands.ResourceId, content *commands.Content, eventMetadata *events.EventMetadata, auditContext *commands.AuditContext) eventstore.EventUnmarshaler {
	e := events.ResourceChanged{
		ResourceId:    resourceId,
		AuditContext:  auditContext,
		Content:       content,
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceChanged{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceChanged); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceRetrievePending(resourceId *commands.ResourceId, resourceInterface string, eventMetadata *events.EventMetadata, auditContext *commands.AuditContext, validUntil time.Time) eventstore.EventUnmarshaler {
	e := events.ResourceRetrievePending{
		ResourceId:        resourceId,
		ResourceInterface: resourceInterface,
		AuditContext:      auditContext,
		EventMetadata:     eventMetadata,
		ValidUntil:        pkgTime.UnixNano(validUntil),
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceRetrievePending{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceRetrievePending); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceRetrieved(resourceId *commands.ResourceId, status commands.Status, content *commands.Content, eventMetadata *events.EventMetadata, auditContext *commands.AuditContext) eventstore.EventUnmarshaler {
	e := events.ResourceRetrieved{
		ResourceId:    resourceId,
		Content:       content,
		Status:        status,
		AuditContext:  auditContext,
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceRetrieved{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceRetrieved); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceDeletePending(resourceId *commands.ResourceId, eventMetadata *events.EventMetadata, auditContext *commands.AuditContext, validUntil time.Time) eventstore.EventUnmarshaler {
	e := events.ResourceDeletePending{
		ResourceId:    resourceId,
		AuditContext:  auditContext,
		EventMetadata: eventMetadata,
		ValidUntil:    pkgTime.UnixNano(validUntil),
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceDeletePending{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceDeletePending); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceDeleted(resourceId *commands.ResourceId, status commands.Status, content *commands.Content, eventMetadata *events.EventMetadata, auditContext *commands.AuditContext) eventstore.EventUnmarshaler {
	e := events.ResourceDeleted{
		ResourceId:    resourceId,
		Content:       content,
		Status:        status,
		AuditContext:  auditContext,
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceDeleted{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceDeleted); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceStateSnapshotTaken(resourceId *commands.ResourceId, latestResourceChange *events.ResourceChanged, eventMetadata *events.EventMetadata, auditContext *commands.AuditContext) eventstore.EventUnmarshaler {
	e := events.NewResourceStateSnapshotTaken()
	e.ResourceId = resourceId
	e.LatestResourceChange = latestResourceChange
	e.EventMetadata = eventMetadata
	e.AuditContext = auditContext

	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceStateSnapshotTaken{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		true,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceStateSnapshotTaken); ok {
				x.CopyData(e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeDeviceMetadataUpdatePending(deviceID string, shadowSync *events.DeviceMetadataUpdatePending_ShadowSynchronization, eventMetadata *events.EventMetadata, auditContext *commands.AuditContext, validUntil time.Time) eventstore.EventUnmarshaler {
	e := events.DeviceMetadataUpdatePending{
		DeviceId:      deviceID,
		UpdatePending: shadowSync,
		EventMetadata: eventMetadata,
		AuditContext:  auditContext,
		ValidUntil:    pkgTime.UnixNano(validUntil),
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.DeviceMetadataUpdatePending{}).EventType(),
		commands.MakeStatusResourceUUID(deviceID),
		e.GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.DeviceMetadataUpdatePending); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeDeviceMetadataUpdated(deviceID string, status *commands.ConnectionStatus, shadowSynchronization commands.ShadowSynchronization, eventMetadata *events.EventMetadata, auditContext *commands.AuditContext, canceled bool) eventstore.EventUnmarshaler {
	e := events.DeviceMetadataUpdated{
		DeviceId:              deviceID,
		Status:                status,
		ShadowSynchronization: shadowSynchronization,
		AuditContext:          auditContext,
		EventMetadata:         eventMetadata,
		Canceled:              canceled,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.DeviceMetadataUpdated{}).EventType(),
		commands.MakeStatusResourceUUID(deviceID),
		e.GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.DeviceMetadataUpdated); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeDeviceMetadata(deviceID string, deviceMetadataUpdated *events.DeviceMetadataUpdated, eventMetadata *events.EventMetadata) eventstore.EventUnmarshaler {
	e := events.DeviceMetadataSnapshotTaken{
		DeviceId:              deviceID,
		DeviceMetadataUpdated: deviceMetadataUpdated,
		EventMetadata:         eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.DeviceMetadataSnapshotTaken{}).EventType(),
		commands.MakeStatusResourceUUID(deviceID),
		e.GetDeviceId(),
		false,
		time.Unix(0, e.GetEventMetadata().GetTimestamp()),
		func(v interface{}) error {
			if x, ok := v.(*events.DeviceMetadataSnapshotTaken); ok {
				x.CopyData(&e)
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

type MockEvent struct {
	VersionI     uint64 `bson:"version"`
	EventTypeI   string `bson:"eventtype"`
	IsSnapshotI  bool   `bson:"issnapshot"`
	AggregateIDI string `bson:"aggregateid"`
	GroupIDI     string `bson:"groupid"`
	DataI        []byte `bson:"data"`
	TimestampI   int64  `bson:"timestamp"`
}

func (e MockEvent) Version() uint64 {
	return e.VersionI
}

func (e MockEvent) EventType() string {
	return e.EventTypeI
}

func (e MockEvent) AggregateID() string {
	return e.AggregateIDI
}

func (e MockEvent) GroupID() string {
	return e.GroupIDI
}

func (e MockEvent) IsSnapshot() bool {
	return e.IsSnapshotI
}

func (e MockEvent) Timestamp() time.Time {
	return time.Unix(0, e.TimestampI)
}

type MockEventHandler struct {
	lock   sync.Mutex
	events map[string]map[string][]eventstore.Event
}

func NewMockEventHandler() *MockEventHandler {
	return &MockEventHandler{events: make(map[string]map[string][]eventstore.Event)}
}

func (eh *MockEventHandler) SetElement(groupID, aggregateID string, e MockEvent) {
	var device map[string][]eventstore.Event
	var ok bool

	eh.lock.Lock()
	defer eh.lock.Unlock()
	if device, ok = eh.events[groupID]; !ok {
		device = make(map[string][]eventstore.Event)
		eh.events[groupID] = device
	}
	device[aggregateID] = append(device[aggregateID], e)
}

func (eh *MockEventHandler) Contains(event eventstore.Event) bool {
	device, ok := eh.events[event.GroupID()]
	if !ok {
		return false
	}
	eventsDB, ok := device[event.AggregateID()]
	if !ok {
		return false
	}

	for _, eventDB := range eventsDB {
		if reflect.DeepEqual(eventDB, event) {
			return true
		}
	}

	return false
}

func (eh *MockEventHandler) ContainsGroupID(groupID string) bool {
	_, ok := eh.events[groupID]
	return ok
}

func (eh *MockEventHandler) Equals(events []eventstore.Event) bool {
	eventsMap := make(map[string]map[string][]eventstore.Event)
	for _, event := range events {
		device, ok := eventsMap[event.GroupID()]
		if !ok {
			device = make(map[string][]eventstore.Event)
			eventsMap[event.GroupID()] = device
		}
		device[event.AggregateID()] = append(device[event.AggregateID()], event)
	}

	if len(eh.events) != len(eventsMap) {
		return false
	}

	// sort slices by version
	for deviceId, resourceEventsMap := range eventsMap {
		for resourceId, resources := range resourceEventsMap {
			sort.Slice(resources, func(i, j int) bool {
				return resources[i].Version() < resources[j].Version()
			})
			eventsMap[deviceId][resourceId] = resources
		}
	}

	return reflect.DeepEqual(eh.events, eventsMap)
}

func (eh *MockEventHandler) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return errors.New("cannot determine type of event")
		}
		var e MockEvent
		err := eu.Unmarshal(&e)
		if err != nil {
			return err
		}
		e.AggregateIDI = eu.AggregateID()
		e.GroupIDI = eu.GroupID()
		e.IsSnapshotI = eu.IsSnapshot()
		if !eu.Timestamp().IsZero() {
			e.TimestampI = eu.Timestamp().UnixNano()
		}
		eh.SetElement(eu.GroupID(), eu.AggregateID(), e)
	}
	return nil
}

func (eh *MockEventHandler) SnapshotEventType() string { return "snapshot" }

func (eh *MockEventHandler) Count() int {
	eh.lock.Lock()
	defer eh.lock.Unlock()
	count := 0
	for _, r := range eh.events {
		for _, e := range r {
			count += len(e)
		}
	}
	return count
}
