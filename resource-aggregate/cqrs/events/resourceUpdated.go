package events

import (
	"github.com/go-ocf/kit/net/http"
	"github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
)

type ResourceUpdated struct {
	pb.ResourceUpdated
}

func (e ResourceUpdated) Version() uint64 {
	return e.EventMetadata.Version
}

func (e ResourceUpdated) Marshal() ([]byte, error) {
	return e.ResourceUpdated.Marshal()
}

func (e *ResourceUpdated) Unmarshal(b []byte) error {
	return e.ResourceUpdated.Unmarshal(b)
}

func (e ResourceUpdated) EventType() string {
	return http.ProtobufContentType(&pb.ResourceUpdated{})
}

func (e ResourceUpdated) AggregateId() string {
	return e.Id
}
