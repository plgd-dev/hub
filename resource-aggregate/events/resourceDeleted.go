package events

import (
	"time"

	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	commands "github.com/plgd-dev/cloud/resource-aggregate/commands"
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
	return e.GetResourceId().ToUUID()
}

func (e *ResourceDeleted) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceDeleted) IsSnapshot() bool {
	return false
}

func (e *ResourceDeleted) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceDeleted) CopyData(event *ResourceDeleted) {
	e.ResourceId = event.GetResourceId()
	e.Status = event.GetStatus()
	e.Content = event.GetContent()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
}

func (e *ResourceDeleted) CheckInitialized() bool {
	return e.GetResourceId() != nil &&
		e.GetStatus() != commands.Status(0) &&
		e.GetContent() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}
