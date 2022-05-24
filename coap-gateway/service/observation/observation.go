package observation

import (
	"context"
	"fmt"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/schema/resources"
	coapMessage "github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

// Query /oic/res resource to get device resource links.
func GetResourceLinks(ctx context.Context, coapConn ClientConn) (schema.ResourceLinks, uint64, error) {
	msg, err := coapConn.Get(ctx, resources.ResourceURI, coapMessage.Option{
		ID:    coapMessage.URIQuery,
		Value: []byte("if=" + interfaces.OC_IF_LL),
	})
	if err != nil {
		return schema.ResourceLinks{}, 0, err
	}
	defer coapConn.ReleaseMessage(msg)

	if msg.Code() != codes.Content {
		return schema.ResourceLinks{}, 0, fmt.Errorf("invalid response code %v", msg.Code())
	}

	data := msg.Body()
	if data == nil {
		return schema.ResourceLinks{}, 0, fmt.Errorf("empty response")
	}

	var links schema.ResourceLinks
	err = cbor.ReadFrom(msg.Body(), &links)
	if err != nil {
		return schema.ResourceLinks{}, 0, err
	}

	return links, msg.Sequence(), nil
}
