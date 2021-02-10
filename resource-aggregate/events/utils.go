package events

import "github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"

//MakeEventMeta for creating EventMetadata for event.
func MakeEventMeta(connectionId string, sequence, version uint64) EventMetadata {
	return EventMetadata{
		ConnectionId: connectionId,
		Sequence:     sequence,
		Version:      version,
		TimestampMs:  utils.TimeNowMs(),
	}
}
