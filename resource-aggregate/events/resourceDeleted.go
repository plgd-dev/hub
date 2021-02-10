package events

import (
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

func (e *ResourceDeleted) AggregateId() string {
	return e.GetResourceId().ToUUID()
}
