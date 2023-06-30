package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceCreated = "resourcecreated"

func (e *ResourceCreated) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceCreated) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceCreated) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceCreated) EventType() string {
	return eventTypeResourceCreated
}

func (e *ResourceCreated) AggregateID() string {
	return e.GetResourceId().ToUUID().String()
}

func (e *ResourceCreated) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceCreated) IsSnapshot() bool {
	return false
}

func (e *ResourceCreated) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceCreated) CopyData(event *ResourceCreated) {
	e.ResourceId = event.GetResourceId()
	e.Status = event.GetStatus()
	e.Content = event.GetContent()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
	e.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
}

func (e *ResourceCreated) CheckInitialized() bool {
	return e.GetResourceId() != nil &&
		e.GetStatus() != commands.Status(0) &&
		e.GetContent() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}
