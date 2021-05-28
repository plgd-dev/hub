package events

import (
	commands "github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/protobuf/proto"
)

const eventTypeDeviceMetadataUpdated = "ocf.cloud.resourceaggregate.events.devicemetadataupdated"

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
