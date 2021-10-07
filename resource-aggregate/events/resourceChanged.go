package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/pkg/time"
	commands "github.com/plgd-dev/hub/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceChanged = "resourcechanged"

func (e *ResourceChanged) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceChanged) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceChanged) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceChanged) EventType() string {
	return eventTypeResourceChanged
}

func (e *ResourceChanged) AggregateID() string {
	return e.GetResourceId().ToUUID()
}

func (e *ResourceChanged) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceChanged) IsSnapshot() bool {
	return false
}

func (e *ResourceChanged) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceChanged) CopyData(event *ResourceChanged) {
	e.ResourceId = event.GetResourceId()
	e.Content = event.GetContent()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
	e.Status = event.GetStatus()
}

func (e *ResourceChanged) CheckInitialized() bool {
	return e.GetResourceId() != nil &&
		e.GetContent() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil &&
		e.GetStatus() != commands.Status(0)
}
