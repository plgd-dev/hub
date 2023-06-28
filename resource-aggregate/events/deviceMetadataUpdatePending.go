package events

import (
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeDeviceMetadataUpdatePending = "devicemetadataupdatepending"

func (d *DeviceMetadataUpdatePending) Version() uint64 {
	return d.GetEventMetadata().GetVersion()
}

func (d *DeviceMetadataUpdatePending) Marshal() ([]byte, error) {
	return proto.Marshal(d)
}

func (d *DeviceMetadataUpdatePending) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, d)
}

func (d *DeviceMetadataUpdatePending) EventType() string {
	return eventTypeDeviceMetadataUpdatePending
}

func (d *DeviceMetadataUpdatePending) AggregateID() string {
	return commands.MakeStatusResourceUUID(d.GetDeviceId()).String()
}

func (d *DeviceMetadataUpdatePending) GroupID() string {
	return d.GetDeviceId()
}

func (d *DeviceMetadataUpdatePending) IsSnapshot() bool {
	return false
}

func (d *DeviceMetadataUpdatePending) Timestamp() time.Time {
	return pkgTime.Unix(0, d.GetEventMetadata().GetTimestamp())
}

func (d *DeviceMetadataUpdatePending) CopyData(event *DeviceMetadataUpdatePending) {
	d.DeviceId = event.GetDeviceId()
	d.UpdatePending = event.GetUpdatePending()
	d.AuditContext = event.GetAuditContext()
	d.EventMetadata = event.GetEventMetadata()
	d.ValidUntil = event.GetValidUntil()
	d.OpenTelemetryCarrier = event.GetOpenTelemetryCarrier()
}

func (d *DeviceMetadataUpdatePending) CheckInitialized() bool {
	return d.GetDeviceId() != "" &&
		d.GetUpdatePending() != nil &&
		d.GetAuditContext() != nil &&
		d.GetEventMetadata() != nil
}

func (d *DeviceMetadataUpdatePending) ValidUntilTime() time.Time {
	return pkgTime.Unix(0, d.GetValidUntil())
}

func (d *DeviceMetadataUpdatePending) IsExpired(now time.Time) bool {
	return IsExpired(now, d.ValidUntilTime())
}
