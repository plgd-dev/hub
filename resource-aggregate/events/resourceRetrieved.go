package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceRetrieved = "resourceretrieved"

func (e *ResourceRetrieved) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceRetrieved) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceRetrieved) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceRetrieved) EventType() string {
	return eventTypeResourceRetrieved
}

func (e *ResourceRetrieved) AggregateID() string {
	return e.GetResourceId().ToUUID().String()
}

func (e *ResourceRetrieved) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceRetrieved) IsSnapshot() bool {
	return false
}

func (e *ResourceRetrieved) ETag() *eventstore.ETagData {
	return nil
}

func (e *ResourceRetrieved) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceRetrieved) ServiceID() (string, bool) {
	return "", false
}

func (e *ResourceRetrieved) Types() []string {
	return e.GetResourceTypes()
}

func (e *ResourceRetrieved) CopyData(event *ResourceRetrieved) {
	e.ResourceId = event.GetResourceId()
	e.Content = event.GetContent()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
	e.Status = event.GetStatus()
	e.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
	e.ResourceTypes = event.GetResourceTypes()
}

func (e *ResourceRetrieved) CheckInitialized() bool {
	return e.GetResourceId() != nil &&
		e.GetContent() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil &&
		e.GetStatus() != commands.Status(0)
}
