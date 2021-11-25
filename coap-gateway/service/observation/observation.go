package observation

import (
	"context"
	"fmt"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/resources"
	coapMessage "github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/coap-gateway/service/message"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

// Query /oic/res resource to determine whether resource with given href is observable
func IsResourceObservable(ctx context.Context, coapConn *tcp.ClientConn, resourceHref, resourceType string) (bool, error) {
	var opts []coapMessage.Option
	if resourceType != "" {
		opts = append(opts, coapMessage.Option{
			ID:    coapMessage.URIQuery,
			Value: []byte("rt=" + resourceType),
		})
	}

	msg, err := coapConn.Get(ctx, resources.ResourceURI, opts...)
	if err != nil {
		return false, err
	}
	defer pool.ReleaseMessage(msg)

	if msg.Code() != codes.Content {
		return false, fmt.Errorf("invalid response code %v", msg.Code())
	}

	message.DecodeMsgToDebug("", msg, "RECEIVED-GET-OIC-RES")
	data := msg.Body()
	if data == nil {
		return false, fmt.Errorf("empty response")
	}

	var links schema.ResourceLinks
	err = cbor.ReadFrom(msg.Body(), &links)
	if err != nil {
		return false, err
	}

	res, ok := links.GetResourceLink(resourceHref)
	if !ok {
		return false, fmt.Errorf("resourceLink for href(%v) not found", resourceHref)
	}

	return res.Policy.BitMask.Has(schema.Observable), nil
}
