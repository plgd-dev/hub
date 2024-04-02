package events

import (
	"time"

	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

// MakeEventMeta for creating EventMetadata for event.
func MakeEventMeta(connectionID string, sequence, version uint64, hubID string) *EventMetadata {
	return &EventMetadata{
		ConnectionId: connectionID,
		Sequence:     sequence,
		Version:      version,
		Timestamp:    time.Now().UnixNano(),
		HubId:        hubID,
	}
}

func EqualResource(x, y *commands.Resource) bool {
	return x.GetDeviceId() == y.GetDeviceId() &&
		EqualStringSlice(x.GetResourceTypes(), y.GetResourceTypes()) &&
		EqualStringSlice(x.GetInterfaces(), y.GetInterfaces()) &&
		x.GetAnchor() == y.GetAnchor() &&
		x.GetTitle() == y.GetTitle() &&
		EqualStringSlice(x.GetSupportedContentTypes(), y.GetSupportedContentTypes()) &&
		x.GetValidUntil() == y.GetValidUntil() &&
		((x.GetPolicy() == nil && y.GetPolicy() == nil) ||
			(x.GetPolicy() != nil && y.GetPolicy() != nil && x.GetPolicy().GetBitFlags() == y.GetPolicy().GetBitFlags()))
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

func IsExpired(now time.Time, validUntil time.Time) bool {
	if validUntil.IsZero() {
		return false
	}
	return now.After(validUntil)
}
