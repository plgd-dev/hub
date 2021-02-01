package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
	"google.golang.org/protobuf/proto"
)

type ResourceRetrieved struct {
	pb.ResourceRetrieved
}

func (e *ResourceRetrieved) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceRetrieved) Marshal() ([]byte, error) {
	return proto.Marshal(&e.ResourceRetrieved)
}

func (e *ResourceRetrieved) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, &e.ResourceRetrieved)
}

func (e *ResourceRetrieved) EventType() string {
	return http.ProtobufContentType(&pb.ResourceRetrieved{})
}

func (e *ResourceRetrieved) AggregateId() string {
	return e.GetId()
}
