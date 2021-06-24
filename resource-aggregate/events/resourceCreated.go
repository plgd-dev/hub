package events

import (
	"time"

	"google.golang.org/protobuf/proto"
)

const eventTypeResourceCreated = "ocf.cloud.resourceaggregate.events.resourcecreated"

func (e *ResourceCreated) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceCreated) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceCreated) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceCreated) EventType() string {
	return eventTypeResourceCreated
}

func (e *ResourceCreated) AggregateID() string {
	return e.GetResourceId().ToUUID()
}

func (e *ResourceCreated) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceCreated) IsSnapshot() bool {
	return false
}

func (e *ResourceCreated) Timestamp() time.Time {
	return time.Unix(0, e.GetEventMetadata().GetTimestamp())
}
