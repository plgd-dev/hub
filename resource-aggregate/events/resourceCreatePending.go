package events

import (
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceCreatePending = "ocf.cloud.resourceaggregate.events.resourcecreatepending"

func (e *ResourceCreatePending) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceCreatePending) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceCreatePending) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceCreatePending) EventType() string {
	return eventTypeResourceCreatePending
}

func (e *ResourceCreatePending) AggregateId() string {
	return e.GetResourceId().ToUUID()
}
