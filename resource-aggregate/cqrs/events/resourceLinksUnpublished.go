package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
	"google.golang.org/protobuf/proto"
)

type ResourceLinksUnpublished struct {
	pb.ResourceLinksUnpublished
}

func (e ResourceLinksUnpublished) Version() uint64 {
	return e.EventMetadata.Version
}

func (e ResourceLinksUnpublished) Marshal() ([]byte, error) {
	return proto.Marshal(&e.ResourceLinksUnpublished)
}

func (e *ResourceLinksUnpublished) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, &e.ResourceLinksUnpublished)
}

func (e ResourceLinksUnpublished) EventType() string {
	return http.ProtobufContentType(&pb.ResourceLinksUnpublished{})
}
