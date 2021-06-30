package utils

import (
	"fmt"
	"time"

	"github.com/golang/snappy"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

func GetDeviceSubject(deviceID string) []string {
	return []string{"events." + deviceID + ".>"}
}

func GetPublishSubject(event eventbus.Event) []string {
	switch event.EventType() {
	case (&events.ResourceLinksPublished{}).EventType(), (&events.ResourceLinksUnpublished{}).EventType(), (&events.ResourceLinksSnapshotTaken{}).EventType():
		return []string{"events." + event.GroupID() + ".resource-links." + event.EventType()}
	case (&events.DeviceMetadataUpdatePending{}).EventType(), (&events.DeviceMetadataUpdated{}).EventType(), (&events.DeviceMetadataSnapshotTaken{}).EventType():
		return []string{"events." + event.GroupID() + ".metadata." + event.EventType()}
	}
	return []string{"events." + event.GroupID() + ".resources." + event.AggregateID() + "." + event.EventType()}
}

func TimeNowMs() uint64 {
	now := time.Now()
	unix := now.UnixNano()
	return uint64(unix / int64(time.Millisecond))
}

type ProtobufMarshaler interface {
	Marshal() ([]byte, error)
}

type ProtobufUnmarshaler interface {
	Unmarshal([]byte) error
}

func Marshal(v interface{}) ([]byte, error) {
	if p, ok := v.(ProtobufMarshaler); ok {
		src, err := p.Marshal()
		if err != nil {
			return nil, fmt.Errorf("cannot marshal event: %w", err)
		}
		dst := make([]byte, 1024)
		return snappy.Encode(dst, src), nil
	}
	return nil, fmt.Errorf("marshal is not supported by %T", v)
}

func Unmarshal(b []byte, v interface{}) error {
	if p, ok := v.(ProtobufUnmarshaler); ok {
		dst := make([]byte, 1024)
		dst, err := snappy.Decode(dst, b)
		if err != nil {
			return fmt.Errorf("cannot decode buffer: %w", err)
		}
		return p.Unmarshal(dst)
	}
	return fmt.Errorf("marshal is not supported by %T", v)
}
