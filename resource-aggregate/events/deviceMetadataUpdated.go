package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeDeviceMetadataUpdated = "devicemetadataupdated"

func (e *DeviceMetadataUpdated) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *DeviceMetadataUpdated) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *DeviceMetadataUpdated) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *DeviceMetadataUpdated) EventType() string {
	return eventTypeDeviceMetadataUpdated
}

func (e *DeviceMetadataUpdated) AggregateID() string {
	return commands.MakeStatusResourceUUID(e.GetDeviceId())
}

func (e *DeviceMetadataUpdated) GroupID() string {
	return e.GetDeviceId()
}

func (e *DeviceMetadataUpdated) IsSnapshot() bool {
	return false
}

func (e *DeviceMetadataUpdated) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *DeviceMetadataUpdated) CopyData(event *DeviceMetadataUpdated) {
	e.DeviceId = event.GetDeviceId()
	e.Status = event.GetStatus()
	e.ShadowSynchronization = event.GetShadowSynchronization()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
	e.Canceled = event.GetCanceled()
	e.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
}

func (e *DeviceMetadataUpdated) CheckInitialized() bool {
	return e.GetDeviceId() != "" &&
		e.GetStatus() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}

// Check if two DeviceMetadataUpdated events are equal
func (e *DeviceMetadataUpdated) Equal(upd *DeviceMetadataUpdated) bool {
	return e.GetStatus().GetValue() == upd.GetStatus().GetValue() &&
		e.GetStatus().GetConnectionId() == upd.GetStatus().GetConnectionId() &&
		e.GetCanceled() == upd.GetCanceled() &&
		e.GetStatus().GetValidUntil() == upd.GetStatus().GetValidUntil() &&
		e.GetShadowSynchronization() == upd.GetShadowSynchronization() &&
		e.GetAuditContext().GetUserId() == upd.GetAuditContext().GetUserId() &&
		e.GetAuditContext().GetCorrelationId() == upd.GetAuditContext().GetCorrelationId()
}
