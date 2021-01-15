package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
	"google.golang.org/protobuf/proto"
)

type ResourceDeletePending struct {
	pb.ResourceDeletePending
}

func (e ResourceDeletePending) Version() uint64 {
	return e.EventMetadata.Version
}

func (e ResourceDeletePending) Marshal() ([]byte, error) {
	return proto.Marshal(&e.ResourceDeletePending)
}

func (e *ResourceDeletePending) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, &e.ResourceDeletePending)
}

func (e ResourceDeletePending) EventType() string {
	return http.ProtobufContentType(&pb.ResourceDeletePending{})
}

func (e ResourceDeletePending) AggregateId() string {
	return e.Id
}
