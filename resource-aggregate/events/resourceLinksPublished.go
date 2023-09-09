package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceLinksPublished = "resourcelinkspublished"

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
	return commands.MakeLinksResourceUUID(e.GetDeviceId()).String()
}

func (e *ResourceLinksPublished) GroupID() string {
	return e.GetDeviceId()
}

func (e *ResourceLinksPublished) IsSnapshot() bool {
	return false
}

func (e *ResourceLinksPublished) ETag() *eventstore.ETagData {
	return nil
}

func (e *ResourceLinksPublished) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceLinksPublished) CopyData(event *ResourceLinksPublished) {
	e.Resources = event.GetResources()
	e.DeviceId = event.GetDeviceId()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
	e.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
}

func (e *ResourceLinksPublished) CheckInitialized() bool {
	return e.GetResources() != nil &&
		e.GetDeviceId() != "" &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}
