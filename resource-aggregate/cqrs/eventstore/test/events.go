package test

import (
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

func MakeResourceLinksPublishedEvent(resources []*commands.Resource, deviceID string, eventMetadata *events.EventMetadata) eventstore.EventUnmarshaler {
	e := events.ResourceLinksPublished{
		Resources: resources,
		DeviceId:  deviceID,
		AuditContext: &commands.AuditContext{
			UserId: "userId",
		},
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceLinksPublished{}).EventType(),
		commands.MakeLinksResourceUUID(e.GetDeviceId()),
		e.GetDeviceId(),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceLinksPublished); ok {
				*x = e
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceLinksUnpublishedEvent(hrefs []string, deviceID string, eventMetadata *events.EventMetadata) eventstore.EventUnmarshaler {
	e := events.ResourceLinksUnpublished{
		Hrefs:    hrefs,
		DeviceId: deviceID,
		AuditContext: &commands.AuditContext{
			UserId: "userId",
		},
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceLinksUnpublished{}).EventType(),
		commands.MakeLinksResourceUUID(e.GetDeviceId()),
		e.GetDeviceId(),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceLinksUnpublished); ok {
				*x = e
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
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceLinksSnapshotTaken); ok {
				*x = *e
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceUpdatePending(resourceId *commands.ResourceId, content *commands.Content, eventMetadata *events.EventMetadata) eventstore.EventUnmarshaler {
	e := events.ResourceUpdatePending{
		ResourceId: resourceId,
		Content:    content,
		AuditContext: &commands.AuditContext{
			UserId: "userId",
		},
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceUpdatePending{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceUpdatePending); ok {
				*x = e
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceUpdated(resourceId *commands.ResourceId, status commands.Status, content *commands.Content, eventMetadata *events.EventMetadata) eventstore.EventUnmarshaler {
	e := events.ResourceUpdated{
		ResourceId: resourceId,
		Content:    content,
		Status:     status,
		AuditContext: &commands.AuditContext{
			UserId: "userId",
		},
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceUpdated{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceUpdated); ok {
				*x = e
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceChangedEvent(resourceId *commands.ResourceId, content *commands.Content, eventMetadata *events.EventMetadata) eventstore.EventUnmarshaler {
	e := events.ResourceChanged{
		ResourceId: resourceId,
		AuditContext: &commands.AuditContext{
			UserId: "userId",
		},
		Content:       content,
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceChanged{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceChanged); ok {
				*x = e
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceRetrievePending(resourceId *commands.ResourceId, resourceInterface string, eventMetadata *events.EventMetadata) eventstore.EventUnmarshaler {
	e := events.ResourceRetrievePending{
		ResourceId:        resourceId,
		ResourceInterface: resourceInterface,
		AuditContext: &commands.AuditContext{
			UserId: "userId",
		},
		EventMetadata: eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceRetrievePending{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceRetrievePending); ok {
				*x = e
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceRetrieved(resourceId *commands.ResourceId, status commands.Status, content *commands.Content, eventMetadata events.EventMetadata) eventstore.EventUnmarshaler {
	e := events.ResourceRetrieved{
		ResourceId: resourceId,
		Content:    content,
		Status:     status,
		AuditContext: &commands.AuditContext{
			UserId: "userId",
		},
		EventMetadata: &eventMetadata,
	}
	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceRetrieved{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceRetrieved); ok {
				*x = e
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}

func MakeResourceStateSnapshotTaken(resourceId *commands.ResourceId, latestResourceChange *events.ResourceChanged, eventMetadata *events.EventMetadata) eventstore.EventUnmarshaler {
	e := events.NewResourceStateSnapshotTaken()
	e.ResourceId = resourceId
	e.LatestResourceChange = latestResourceChange
	e.EventMetadata = eventMetadata

	return eventstore.NewLoadedEvent(
		e.GetEventMetadata().GetVersion(),
		(&events.ResourceStateSnapshotTaken{}).EventType(),
		e.GetResourceId().ToUUID(),
		e.GetResourceId().GetDeviceId(),
		func(v interface{}) error {
			if x, ok := v.(*events.ResourceStateSnapshotTaken); ok {
				*x = *e
				return nil
			}
			return fmt.Errorf("cannot unmarshal event")
		},
	)
}
