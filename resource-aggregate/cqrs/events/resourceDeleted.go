package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
)

type ResourceDeleted struct {
	pb.ResourceDeleted
}

func (e ResourceDeleted) Version() uint64 {
	return e.EventMetadata.Version
}

func (e ResourceDeleted) Marshal() ([]byte, error) {
	return e.ResourceDeleted.Marshal()
}

func (e *ResourceDeleted) Unmarshal(b []byte) error {
	return e.ResourceDeleted.Unmarshal(b)
}

func (e ResourceDeleted) EventType() string {
	return http.ProtobufContentType(&pb.ResourceDeleted{})
}

func (e ResourceDeleted) AggregateId() string {
	return e.Id
}
