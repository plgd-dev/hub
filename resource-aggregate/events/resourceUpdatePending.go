package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceUpdatePending = "resourceupdatepending"

func (e *ResourceUpdatePending) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceUpdatePending) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceUpdatePending) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceUpdatePending) EventType() string {
	return eventTypeResourceUpdatePending
}

func (e *ResourceUpdatePending) AggregateID() string {
	return e.GetResourceId().ToUUID()
}

func (e *ResourceUpdatePending) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceUpdatePending) IsSnapshot() bool {
	return false
}

func (e *ResourceUpdatePending) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceUpdatePending) CopyData(event *ResourceUpdatePending) {
	e.ResourceId = event.GetResourceId()
	e.ResourceInterface = event.GetResourceInterface()
	e.Content = event.GetContent()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
	e.ValidUntil = event.GetValidUntil()
	e.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
}

func (e *ResourceUpdatePending) CheckInitialized() bool {
	return e.GetResourceId() != nil &&
		e.GetResourceInterface() != "" &&
		e.GetContent() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}

func (e *ResourceUpdatePending) ValidUntilTime() time.Time {
	return pkgTime.Unix(0, e.GetValidUntil())
}

func (e *ResourceUpdatePending) IsExpired(now time.Time) bool {
	return IsExpired(now, e.ValidUntilTime())
}
