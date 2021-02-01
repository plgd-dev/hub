package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
	"google.golang.org/protobuf/proto"
)

type ResourceChanged struct {
	pb.ResourceChanged
}

func (e *ResourceChanged) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceChanged) Marshal() ([]byte, error) {
	return proto.Marshal(&e.ResourceChanged)
}

func (e *ResourceChanged) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, &e.ResourceChanged)
}

func (e *ResourceChanged) EventType() string {
	return http.ProtobufContentType(&pb.ResourceChanged{})
}

func (e *ResourceChanged) AggregateId() string {
	return e.GetId()
}
