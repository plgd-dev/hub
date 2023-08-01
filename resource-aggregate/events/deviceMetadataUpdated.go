package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
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
	return commands.MakeStatusResourceUUID(d.GetDeviceId()).String()
}

func (d *DeviceMetadataUpdated) GroupID() string {
	return d.GetDeviceId()
}

func (d *DeviceMetadataUpdated) IsSnapshot() bool {
	return false
}

func (d *DeviceMetadataUpdated) ETag() *eventstore.ETagData {
	return nil
}

func (d *DeviceMetadataUpdated) Timestamp() time.Time {
	return pkgTime.Unix(0, d.GetEventMetadata().GetTimestamp())
}

func (d *DeviceMetadataUpdated) CopyData(event *DeviceMetadataUpdated) {
	d.DeviceId = event.GetDeviceId()
	d.Connection = event.GetConnection()
	d.TwinSynchronization = event.GetTwinSynchronization()
	d.TwinEnabled = event.GetTwinEnabled()
	d.AuditContext = event.GetAuditContext()
	d.EventMetadata = event.GetEventMetadata()
	d.Canceled = event.GetCanceled()
	d.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
}

func (d *DeviceMetadataUpdated) CheckInitialized() bool {
	return d.GetDeviceId() != "" &&
		d.GetConnection() != nil &&
		d.GetAuditContext() != nil &&
		d.GetEventMetadata() != nil
}

// Equal checks if two DeviceMetadataUpdated events are equal.
func (d *DeviceMetadataUpdated) Equal(upd *DeviceMetadataUpdated) bool {
	return d.GetConnection().GetStatus() == upd.GetConnection().GetStatus() &&
		d.GetConnection().GetId() == upd.GetConnection().GetId() &&
		d.GetConnection().GetOnlineValidUntil() == upd.GetConnection().GetOnlineValidUntil() &&
		d.GetConnection().GetProtocol() == upd.GetConnection().GetProtocol() &&
		d.GetCanceled() == upd.GetCanceled() &&
		d.GetTwinEnabled() == upd.GetTwinEnabled() &&
		d.GetAuditContext().GetUserId() == upd.GetAuditContext().GetUserId() &&
		d.GetAuditContext().GetCorrelationId() == upd.GetAuditContext().GetCorrelationId() &&
		d.GetTwinSynchronization().GetCommandMetadata().GetConnectionId() == upd.GetTwinSynchronization().GetCommandMetadata().GetConnectionId() &&
		d.GetTwinSynchronization().GetSyncingAt() == upd.GetTwinSynchronization().GetSyncingAt() &&
		d.GetTwinSynchronization().GetInSyncAt() == upd.GetTwinSynchronization().GetInSyncAt() &&
		d.GetTwinSynchronization().GetState() == upd.GetTwinSynchronization().GetState() &&
		d.GetTwinSynchronization().GetForceResynchronizationAt() == upd.GetTwinSynchronization().GetForceResynchronizationAt()
}
