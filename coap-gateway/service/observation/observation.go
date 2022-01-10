package observation

import (
	"context"
	"fmt"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/resources"
	coapMessage "github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	pkgStrings "github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

// Query /oic/res resource to determine whether resource with given href is observable and supports given interface.
func IsResourceObservableWithInterface(ctx context.Context, coapConn *tcp.ClientConn, resourceHref, resourceType, observeInterface string) (bool, error) {
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
	defer coapConn.ReleaseMessage(msg)

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

	observable := res.Policy.BitMask.Has(schema.Observable)
	if observeInterface == "" || !observable {
		return observable, nil
	}

	return pkgStrings.Contains(res.Interfaces, observeInterface), nil
}
