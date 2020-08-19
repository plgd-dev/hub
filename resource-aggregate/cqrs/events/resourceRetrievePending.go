package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
)

type ResourceRetrievePending struct {
	pb.ResourceRetrievePending
}

func (e ResourceRetrievePending) Version() uint64 {
	return e.EventMetadata.Version
}

func (e ResourceRetrievePending) Marshal() ([]byte, error) {
	return e.ResourceRetrievePending.Marshal()
}

func (e *ResourceRetrievePending) Unmarshal(b []byte) error {
	return e.ResourceRetrievePending.Unmarshal(b)
}

func (e ResourceRetrievePending) EventType() string {
	return http.ProtobufContentType(&pb.ResourceRetrievePending{})
}

func (e ResourceRetrievePending) AggregateId() string {
	return e.Id
}
