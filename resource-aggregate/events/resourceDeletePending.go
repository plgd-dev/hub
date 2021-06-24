package events

import (
	"time"

	"google.golang.org/protobuf/proto"
)

const eventTypeResourceDeletePending = "ocf.cloud.resourceaggregate.events.resourcedeletepending"

func (e *ResourceDeletePending) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceDeletePending) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceDeletePending) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceDeletePending) EventType() string {
	return eventTypeResourceDeletePending
}

func (e *ResourceDeletePending) AggregateID() string {
	return e.GetResourceId().ToUUID()
}

func (e *ResourceDeletePending) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceDeletePending) IsSnapshot() bool {
	return false
}

func (e *ResourceDeletePending) Timestamp() time.Time {
	return time.Unix(0, e.GetEventMetadata().GetTimestamp())
}
