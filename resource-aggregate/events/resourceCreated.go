package events

import (
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

func (e *ResourceCreated) AggregateId() string {
	return e.GetResourceId().ToUUID()
}
