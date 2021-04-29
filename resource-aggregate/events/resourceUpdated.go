package events

import (
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceUpdated = "ocf.cloud.resourceaggregate.events.resourceupdated"

func (e *ResourceUpdated) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceUpdated) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceUpdated) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceUpdated) EventType() string {
	return eventTypeResourceUpdated
}

func (e *ResourceUpdated) AggregateID() string {
	return e.GetResourceId().ToUUID()
}

func (e *ResourceUpdated) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceUpdated) IsSnapshot() bool {
	return false
}
