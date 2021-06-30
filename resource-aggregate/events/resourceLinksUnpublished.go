package events

import (
	"time"

	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceLinksUnpublished = "resourcelinksunpublished"

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

func (e *ResourceLinksUnpublished) AggregateID() string {
	return commands.MakeLinksResourceUUID(e.GetDeviceId())
}

func (e *ResourceLinksUnpublished) GroupID() string {
	return e.GetDeviceId()
}

func (e *ResourceLinksUnpublished) IsSnapshot() bool {
	return false
}

func (e *ResourceLinksUnpublished) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}
