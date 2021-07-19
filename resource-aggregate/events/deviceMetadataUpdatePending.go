package events

import (
	"time"

	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	commands "github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeDeviceMetadataUpdatePending = "devicemetadataupdatepending"

func (e *DeviceMetadataUpdatePending) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *DeviceMetadataUpdatePending) Marshal() ([]byte, error) {
	return proto.Marshal(e)
}

func (e *DeviceMetadataUpdatePending) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, e)
}

func (e *DeviceMetadataUpdatePending) EventType() string {
	return eventTypeDeviceMetadataUpdatePending
}

func (e *DeviceMetadataUpdatePending) AggregateID() string {
	return commands.MakeStatusResourceUUID(e.GetDeviceId())
}

func (e *DeviceMetadataUpdatePending) GroupID() string {
	return e.GetDeviceId()
}

func (e *DeviceMetadataUpdatePending) IsSnapshot() bool {
	return false
}

func (e *DeviceMetadataUpdatePending) Timestamp() time.Time {
	return pkgTime.Unix(0, e.GetEventMetadata().GetTimestamp())
}

func (e *DeviceMetadataUpdatePending) CopyData(event *DeviceMetadataUpdatePending) {
	e.DeviceId = event.GetDeviceId()
	e.UpdatePending = event.GetUpdatePending()
	e.AuditContext = event.GetAuditContext()
	e.EventMetadata = event.GetEventMetadata()
}

func (e *DeviceMetadataUpdatePending) CheckInitialized() bool {
	return e.GetDeviceId() != "" &&
		e.GetUpdatePending() != nil &&
		e.GetAuditContext() != nil &&
		e.GetEventMetadata() != nil
}
