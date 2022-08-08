package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeDeviceMetadataUpdated = "devicemetadataupdated"

func (d *DeviceMetadataUpdated) Version() uint64 {
	return d.GetEventMetadata().GetVersion()
}

func (d *DeviceMetadataUpdated) Marshal() ([]byte, error) {
	return proto.Marshal(d)
}

func (d *DeviceMetadataUpdated) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, d)
}

func (d *DeviceMetadataUpdated) EventType() string {
	return eventTypeDeviceMetadataUpdated
}

func (d *DeviceMetadataUpdated) AggregateID() string {
	return commands.MakeStatusResourceUUID(d.GetDeviceId())
}

func (d *DeviceMetadataUpdated) GroupID() string {
	return d.GetDeviceId()
}

func (d *DeviceMetadataUpdated) IsSnapshot() bool {
	return false
}

func (d *DeviceMetadataUpdated) Timestamp() time.Time {
	return pkgTime.Unix(0, d.GetEventMetadata().GetTimestamp())
}

func (d *DeviceMetadataUpdated) CopyData(event *DeviceMetadataUpdated) {
	d.DeviceId = event.GetDeviceId()
	d.Status = event.GetStatus()
	d.ShadowSynchronization = event.GetShadowSynchronization()
	d.AuditContext = event.GetAuditContext()
	d.EventMetadata = event.GetEventMetadata()
	d.Canceled = event.GetCanceled()
	d.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
}

func (d *DeviceMetadataUpdated) CheckInitialized() bool {
	return d.GetDeviceId() != "" &&
		d.GetStatus() != nil &&
		d.GetAuditContext() != nil &&
		d.GetEventMetadata() != nil
}

// Equal checks if two DeviceMetadataUpdated events are equal.
func (d *DeviceMetadataUpdated) Equal(upd *DeviceMetadataUpdated) bool {
	return d.GetStatus().GetValue() == upd.GetStatus().GetValue() &&
		d.GetStatus().GetConnectionId() == upd.GetStatus().GetConnectionId() &&
		d.GetCanceled() == upd.GetCanceled() &&
		d.GetStatus().GetValidUntil() == upd.GetStatus().GetValidUntil() &&
		d.GetShadowSynchronization() == upd.GetShadowSynchronization() &&
		d.GetAuditContext().GetUserId() == upd.GetAuditContext().GetUserId() &&
		d.GetAuditContext().GetCorrelationId() == upd.GetAuditContext().GetCorrelationId()
}
