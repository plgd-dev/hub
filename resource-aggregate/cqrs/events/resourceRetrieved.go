package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
)

type ResourceRetrieved struct {
	pb.ResourceRetrieved
}

func (e ResourceRetrieved) Version() uint64 {
	return e.EventMetadata.Version
}

func (e ResourceRetrieved) Marshal() ([]byte, error) {
	return e.ResourceRetrieved.Marshal()
}

func (e *ResourceRetrieved) Unmarshal(b []byte) error {
	return e.ResourceRetrieved.Unmarshal(b)
}

func (e ResourceRetrieved) EventType() string {
	return http.ProtobufContentType(&pb.ResourceRetrieved{})
}

func (e ResourceRetrieved) AggregateId() string {
	return e.Id
}
