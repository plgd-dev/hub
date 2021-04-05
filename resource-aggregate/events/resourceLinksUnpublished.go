package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceLinksUnpublished = "ocf.cloud.resourceaggregate.events.resourcelinksunpublished"

func (e *ResourceLinksUnpublished) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceLinksUnpublished) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceLinksUnpublished) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceLinksUnpublished) EventType() string {
	return eventTypeResourceLinksUnpublished
}

func (e *ResourceLinksUnpublished) AggregateId() string {
	return commands.MakeLinksResourceUUID(e.GetDeviceId())
}
