package events

import (
	"time"

	"google.golang.org/protobuf/proto"
)

const eventTypeResourceRetrievePending = "ocf.cloud.resourceaggregate.events.resourceretrievepending"

func (e *ResourceRetrievePending) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceRetrievePending) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceRetrievePending) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceRetrievePending) EventType() string {
	return eventTypeResourceRetrievePending
}

func (e *ResourceRetrievePending) AggregateID() string {
	return e.GetResourceId().ToUUID()
}

func (e *ResourceRetrievePending) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceRetrievePending) IsSnapshot() bool {
	return false
}

func (e *ResourceRetrievePending) Timestamp() time.Time {
	return time.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceRetrievePending) CopyData(event *ResourceRetrievePending) {
	e.ResourceId = event.GetResourceId()
	e.ResourceInterface = event.GetResourceInterface()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
}
