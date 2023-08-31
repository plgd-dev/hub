package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
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
	return e.GetResourceId().ToUUID().String()
}

func (e *ResourceUpdated) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceUpdated) IsSnapshot() bool {
	return false
}

func (e *ResourceUpdated) ETag() *eventstore.ETagData {
	return nil
}

func (e *ResourceUpdated) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceUpdated) ServiceID() (string, bool) {
	return "", false
}

func (e *ResourceUpdated) CopyData(event *ResourceUpdated) {
	e.ResourceId = event.GetResourceId()
	e.Status = event.GetStatus()
	e.Content = event.GetContent()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
	e.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
}

func (e *ResourceUpdated) CheckInitialized() bool {
	return e.GetResourceId() != nil &&
		e.GetStatus() != commands.Status(0) &&
		e.GetContent() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}
