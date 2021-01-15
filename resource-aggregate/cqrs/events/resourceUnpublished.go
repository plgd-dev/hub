package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
	"google.golang.org/protobuf/proto"
)

type ResourceUnpublished struct {
	pb.ResourceUnpublished
}

func (e ResourceUnpublished) Version() uint64 {
	return e.EventMetadata.Version
}

func (e ResourceUnpublished) Marshal() ([]byte, error) {
	return proto.Marshal(&e.ResourceUnpublished)
}

func (e *ResourceUnpublished) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, &e.ResourceUnpublished)
}

func (e ResourceUnpublished) EventType() string {
	return http.ProtobufContentType(&pb.ResourceUnpublished{})
}

func (e ResourceUnpublished) AggregateId() string {
	return e.Id
}
