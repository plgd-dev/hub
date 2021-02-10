package events

import (
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceRetrieved = "ocf.cloud.resourceaggregate.events.resourceretrieved"

func (e *ResourceRetrieved) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceRetrieved) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceRetrieved) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceRetrieved) EventType() string {
	return eventTypeResourceRetrieved
}

func (e *ResourceRetrieved) AggregateId() string {
	return e.GetResourceId().ToUUID()
}
