package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
	"google.golang.org/protobuf/proto"
)

type ResourceRetrievePending struct {
	pb.ResourceRetrievePending
}

func (e *ResourceRetrievePending) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceRetrievePending) Marshal() ([]byte, error) {
	return proto.Marshal(&e.ResourceRetrievePending)
}

func (e *ResourceRetrievePending) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, &e.ResourceRetrievePending)
}

func (e *ResourceRetrievePending) EventType() string {
	return http.ProtobufContentType(&pb.ResourceRetrievePending{})
}

func (e *ResourceRetrievePending) AggregateId() string {
	return e.GetId()
}
