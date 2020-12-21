package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
	"google.golang.org/protobuf/proto"
)

type ResourceUpdatePending struct {
	pb.ResourceUpdatePending
}

func (e ResourceUpdatePending) Version() uint64 {
	return e.EventMetadata.Version
}

func (e ResourceUpdatePending) Marshal() ([]byte, error) {
	return proto.Marshal(&e.ResourceUpdatePending)
}

func (e *ResourceUpdatePending) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, &e.ResourceUpdatePending)
}

func (e ResourceUpdatePending) EventType() string {
	return http.ProtobufContentType(&pb.ResourceUpdatePending{})
}

func (e ResourceUpdatePending) AggregateId() string {
	return e.Id
}
