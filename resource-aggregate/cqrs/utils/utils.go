package utils

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/snappy"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
)

func GetTopics(deviceId string) []string {
	return []string{"events-" + deviceId + "-resource-aggregate"}
}

func MakeResourceId(deviceID, href string) string {
	return uuid.NewV5(uuid.NamespaceURL, deviceID+href).String()
}

func TimeNowMs() uint64 {
	now := time.Now()
	unix := now.UnixNano()
	return uint64(unix / int64(time.Millisecond))
}

//CreateEventMeta for creating EventMetadata for event.
func MakeEventMeta(connectionId string, sequence, version uint64) pb.EventMetadata {
	return pb.EventMetadata{
		ConnectionId: connectionId,
		Sequence:     sequence,
		Version:      version,
		TimestampMs:  TimeNowMs(),
	}
}

func MakeAuditContext(deviceID, userID, correlationId string) pb.AuditContext {
	return pb.AuditContext{
		UserId:        userID,
		DeviceId:      deviceID,
		CorrelationId: correlationId,
	}
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
