package events

import (
	"time"

	"google.golang.org/protobuf/proto"
)

const eventTypeResourceChanged = "ocf.cloud.resourceaggregate.events.resourcechanged"

func (e *ResourceChanged) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceChanged) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceChanged) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceChanged) EventType() string {
	return eventTypeResourceChanged
}

func (e *ResourceChanged) AggregateID() string {
	return e.GetResourceId().ToUUID()
}

func (e *ResourceChanged) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceChanged) IsSnapshot() bool {
	return false
}

func (e *ResourceChanged) Timestamp() time.Time {
	return time.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceChanged) Clone() *ResourceChanged {
	if e == nil {
		return nil
	}
	return &ResourceChanged{
		ResourceId:    e.GetResourceId(),
		Content:       e.GetContent(),
		AuditContext:  e.GetAuditContext(),
		EventMetadata: e.GetEventMetadata(),
		Status:        e.GetStatus(),
	}
}
