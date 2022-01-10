package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceCreatePending = "resourcecreatepending"

func (e *ResourceCreatePending) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceCreatePending) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *ResourceCreatePending) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *ResourceCreatePending) EventType() string {
	return eventTypeResourceCreatePending
}

func (e *ResourceCreatePending) AggregateID() string {
	return e.GetResourceId().ToUUID()
}

func (e *ResourceCreatePending) GroupID() string {
	return e.GetResourceId().GetDeviceId()
}

func (e *ResourceCreatePending) IsSnapshot() bool {
	return false
}

func (e *ResourceCreatePending) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *ResourceCreatePending) CopyData(event *ResourceCreatePending) {
	e.ResourceId = event.GetResourceId()
	e.Content = event.GetContent()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
	e.ValidUntil = event.GetValidUntil()
}

func (e *ResourceCreatePending) CheckInitialized() bool {
	return e.GetResourceId() != nil &&
		e.GetContent() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}

func (e *ResourceCreatePending) ValidUntilTime() time.Time {
	return pkgTime.Unix(0, e.GetValidUntil())
}

func (e *ResourceCreatePending) IsExpired(now time.Time) bool {
	return IsExpired(now, e.ValidUntilTime())
}
