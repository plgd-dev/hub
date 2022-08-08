package events

import (
	"bytes"
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeResourceChanged = "resourcechanged"

func (rc *ResourceChanged) Version() uint64 {
	return rc.GetEventMetadata().GetVersion()
}

func (rc *ResourceChanged) Marshal() ([]byte, error) {
	return proto.Marshal(rc)
}

func (rc *ResourceChanged) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, rc)
}

func (rc *ResourceChanged) EventType() string {
	return eventTypeResourceChanged
}

func (rc *ResourceChanged) AggregateID() string {
	return rc.GetResourceId().ToUUID()
}

func (rc *ResourceChanged) GroupID() string {
	return rc.GetResourceId().GetDeviceId()
}

func (rc *ResourceChanged) IsSnapshot() bool {
	return false
}

func (rc *ResourceChanged) Timestamp() time.Time {
	return pkgTime.Unix(0, rc.GetEventMetadata().GetTimestamp())
}

func (rc *ResourceChanged) CopyData(event *ResourceChanged) {
	rc.ResourceId = event.GetResourceId()
	rc.Content = event.GetContent()
	rc.AuditContext = event.GetAuditContext()
	rc.EventMetadata = event.GetEventMetadata()
	rc.Status = event.GetStatus()
	rc.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
}

func (rc *ResourceChanged) CheckInitialized() bool {
	return rc.GetResourceId() != nil &&
		rc.GetContent() != nil &&
		rc.GetAuditContext() != nil &&
		rc.GetEventMetadata() != nil &&
		rc.GetStatus() != commands.Status(0)
}

func (rc *ResourceChanged) Equal(changed *ResourceChanged) bool {
	if rc.GetStatus() != changed.GetStatus() {
		return false
	}

	if rc.GetContent().GetCoapContentFormat() != changed.GetContent().GetCoapContentFormat() ||
		rc.GetContent().GetContentType() != changed.GetContent().GetContentType() ||
		!bytes.Equal(rc.GetContent().GetData(), changed.GetContent().GetData()) {
		return false
	}

	if rc.GetAuditContext().GetUserId() != changed.GetAuditContext().GetUserId() {
		return false
	}

	return true
}
