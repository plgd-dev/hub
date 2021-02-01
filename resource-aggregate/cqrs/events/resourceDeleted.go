package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
	"google.golang.org/protobuf/proto"
)

type ResourceDeleted struct {
	pb.ResourceDeleted
}

func (e *ResourceDeleted) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceDeleted) Marshal() ([]byte, error) {
	return proto.Marshal(&e.ResourceDeleted)
}

func (e *ResourceDeleted) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, &e.ResourceDeleted)
}

func (e *ResourceDeleted) EventType() string {
	return http.ProtobufContentType(&pb.ResourceDeleted{})
}

func (e *ResourceDeleted) AggregateId() string {
	return e.GetId()
}
