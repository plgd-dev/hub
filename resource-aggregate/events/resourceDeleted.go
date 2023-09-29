package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceDeleted = "resourcedeleted"

func (e *ResourceDeleted) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceDeleted) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceDeleted) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceDeleted) EventType() string {
	return eventTypeResourceDeleted
}

func (e *ResourceDeleted) AggregateID() string {
	return e.GetResourceId().ToUUID().String()
}

func (e *ResourceDeleted) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceDeleted) IsSnapshot() bool {
	return false
}

func (e *ResourceDeleted) ETag() *eventstore.ETagData {
	return nil
}

func (e *ResourceDeleted) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceDeleted) ServiceID() (string, bool) {
	return "", false
}

func (e *ResourceDeleted) CopyData(event *ResourceDeleted) {
	e.ResourceId = event.GetResourceId()
	e.Status = event.GetStatus()
	e.Content = event.GetContent()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
	e.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
}

func (e *ResourceDeleted) CheckInitialized() bool {
	return e.GetResourceId() != nil &&
		e.GetStatus() != commands.Status(0) &&
		e.GetContent() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}
