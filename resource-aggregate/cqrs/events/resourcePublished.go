package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
	"google.golang.org/protobuf/proto"
)

type ResourcePublished struct {
	pb.ResourcePublished
}

func (e ResourcePublished) Version() uint64 {
	return e.EventMetadata.Version
}

func (e ResourcePublished) Marshal() ([]byte, error) {
	return proto.Marshal(&e.ResourcePublished)
}

func (e *ResourcePublished) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, &e.ResourcePublished)
}

func (e ResourcePublished) EventType() string {
	return http.ProtobufContentType(&pb.ResourcePublished{})
}

func (e ResourcePublished) AggregateId() string {
	return e.Id
}
