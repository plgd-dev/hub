package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceDeletePending = "resourcedeletepending"

func (e *ResourceDeletePending) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceDeletePending) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceDeletePending) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceDeletePending) EventType() string {
	return eventTypeResourceDeletePending
}

func (e *ResourceDeletePending) AggregateID() string {
	return e.GetResourceId().ToUUID().String()
}

func (e *ResourceDeletePending) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceDeletePending) IsSnapshot() bool {
	return false
}

func (e *ResourceDeletePending) ETag() *eventstore.ETagData {
	return nil
}

func (e *ResourceDeletePending) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceDeletePending) ServiceID() (string, bool) {
	return "", false
}

func (e *ResourceDeletePending) CopyData(event *ResourceDeletePending) {
	e.ResourceId = event.GetResourceId()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
	e.ValidUntil = event.GetValidUntil()
	e.ResourceInterface = event.GetResourceInterface()
	e.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
}

func (e *ResourceDeletePending) CheckInitialized() bool {
	return e.GetResourceId() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}

func (e *ResourceDeletePending) ValidUntilTime() time.Time {
	return pkgTime.Unix(0, e.GetValidUntil())
}

func (e *ResourceDeletePending) IsExpired(now time.Time) bool {
	return IsExpired(now, e.ValidUntilTime())
}
