package events

import (
	"time"

	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	commands "github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceUpdated = "resourceupdated"

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

func (e *ResourceUpdated) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceUpdated) CopyData(event *ResourceUpdated) {
	e.ResourceId = event.GetResourceId()
	e.Status = event.GetStatus()
	e.Content = event.GetContent()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
}

func (e *ResourceUpdated) CheckInitialized() bool {
	return e.GetResourceId() != nil &&
		e.GetStatus() != commands.Status(0) &&
		e.GetContent() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}
