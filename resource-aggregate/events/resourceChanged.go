package events

import (
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

func (e *ResourceChanged) AggregateId() string {
	return e.GetResourceId().ToUUID()
}
