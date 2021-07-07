package events

import (
	"time"

	"google.golang.org/protobuf/proto"
)

const eventTypeResourceDeleted = "ocf.cloud.resourceaggregate.events.resourcedeleted"

func (e *ResourceDeleted) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceDeleted) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceDeleted) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceDeleted) EventType() string {
	return eventTypeResourceDeleted
}

func (e *ResourceDeleted) AggregateID() string {
	return e.GetResourceId().ToUUID()
}

func (e *ResourceDeleted) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceDeleted) IsSnapshot() bool {
	return false
}

func (e *ResourceDeleted) Timestamp() time.Time {
	return time.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceDeleted) CopyData(event *ResourceDeleted) {
	e.ResourceId = event.GetResourceId()
	e.Status = event.GetStatus()
	e.Content = event.GetContent()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
}
