package events

import (
	commands "github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
)

//MakeEventMeta for creating EventMetadata for event.
func MakeEventMeta(connectionId string, sequence, version uint64) *EventMetadata {
	return &EventMetadata{
		ConnectionId: connectionId,
		Sequence:     sequence,
		Version:      version,
		TimestampMs:  utils.TimeNowMs(),
	}
}

func EqualResource(x, y *commands.Resource) bool {
	return x.DeviceId == y.DeviceId &&
		EqualStringSlice(x.ResourceTypes, y.ResourceTypes) &&
		EqualStringSlice(x.Interfaces, y.Interfaces) &&
		x.Anchor == y.Anchor &&
		x.Title == y.Title &&
		EqualStringSlice(x.SupportedContentTypes, y.SupportedContentTypes) &&
		x.ValidUntil == y.ValidUntil &&
		((x.Policies == nil && y.Policies == nil) ||
			(x.Policies != nil && y.Policies != nil && x.Policies.BitFlags == y.Policies.BitFlags))
}

func EqualStringSlice(x, y []string) bool {
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
