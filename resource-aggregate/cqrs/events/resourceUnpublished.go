package events

import (
	"github.com/go-ocf/kit/net/http"
	"github.com/go-ocf/cloud/resource-aggregate/pb"
)

type ResourceUnpublished struct {
	pb.ResourceUnpublished
}

func (e ResourceUnpublished) Version() uint64 {
	return e.EventMetadata.Version
}

func (e ResourceUnpublished) Marshal() ([]byte, error) {
	return e.ResourceUnpublished.Marshal()
}

func (e *ResourceUnpublished) Unmarshal(b []byte) error {
	return e.ResourceUnpublished.Unmarshal(b)
}

func (e ResourceUnpublished) EventType() string {
	return http.ProtobufContentType(&pb.ResourceUnpublished{})
}

func (e ResourceUnpublished) AggregateId() string {
	return e.Id
}
