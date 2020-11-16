package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
)

type ResourceDeletePending struct {
	pb.ResourceDeletePending
}

func (e ResourceDeletePending) Version() uint64 {
	return e.EventMetadata.Version
}

func (e ResourceDeletePending) Marshal() ([]byte, error) {
	return e.ResourceDeletePending.Marshal()
}

func (e *ResourceDeletePending) Unmarshal(b []byte) error {
	return e.ResourceDeletePending.Unmarshal(b)
}

func (e ResourceDeletePending) EventType() string {
	return http.ProtobufContentType(&pb.ResourceDeletePending{})
}

func (e ResourceDeletePending) AggregateId() string {
	return e.Id
}
