package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceLinksPublished = "ocf.cloud.resourceaggregate.events.resourcelinkspublished"

func (e *ResourceLinksPublished) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceLinksPublished) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceLinksPublished) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceLinksPublished) EventType() string {
	return eventTypeResourceLinksPublished
}

func (e *ResourceLinksPublished) AggregateID() string {
	return commands.MakeLinksResourceUUID(e.GetDeviceId())
}

func (e *ResourceLinksPublished) GroupID() string {
	return e.GetDeviceId()
}

func (e *ResourceLinksPublished) IsSnapshot() bool {
	return false
}
