package events

import (
	commands "github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeDeviceMetadataUpdatePending = "ocf.cloud.resourceaggregate.events.devicemetadataupdatepending"

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
