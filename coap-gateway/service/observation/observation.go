package observation

import (
	"context"
	"fmt"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	coapMessage "github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

// Query resource links from the given resource with the interface oic.if.ll.
func GetResourceLinks(ctx context.Context, coapConn ClientConn, href string) (schema.ResourceLinks, uint64, error) {
	msg, err := coapConn.Get(ctx, href, coapMessage.Option{
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
