package events

import (
	"time"

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
	return time.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceLinksUnpublished) CopyData(event *ResourceLinksUnpublished) {
	e.Hrefs = event.GetHrefs()
	e.DeviceId = event.GetDeviceId()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
}
