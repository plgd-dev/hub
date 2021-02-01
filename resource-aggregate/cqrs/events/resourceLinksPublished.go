package events

import (
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/net/http"
	"google.golang.org/protobuf/proto"
)

type ResourceLinksPublished struct {
	pb.ResourceLinksPublished
}

func (e *ResourceLinksPublished) Version() uint64 {
	return e.GetEventMetadata().GetVersion()
}

func (e *ResourceLinksPublished) Marshal() ([]byte, error) {
	return proto.Marshal(&e.ResourceLinksPublished)
}

func (e *ResourceLinksPublished) Unmarshal(b []byte) error {
	return proto.Unmarshal(b, &e.ResourceLinksPublished)
}

func (e *ResourceLinksPublished) EventType() string {
	return http.ProtobufContentType(&pb.ResourceLinksPublished{})
}

func (e *ResourceLinksPublished) AggregateId() string {
	return utils.MakeResourceId(e.GetDeviceId(), utils.ResourceLinksHref)
}
