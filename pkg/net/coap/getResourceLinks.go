package coap

import (
	"context"
	"fmt"
	"net"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

type ClientConn = interface {
	Get(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error)
	ReleaseMessage(m *pool.Message)
	RemoteAddr() net.Addr
}

// GetResourceLinks queries the resource links from the given resource.
func GetResourceLinks(ctx context.Context, coapConn ClientConn, href string, opts ...message.Option) (schema.ResourceLinks, uint64, error) {
	msg, err := coapConn.Get(ctx, href, opts...)
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

// GetResourceLinksWithLinkInterface query resource links from the given resource with the interface oic.if.ll.
func GetResourceLinksWithLinkInterface(ctx context.Context, coapConn ClientConn, href string) (schema.ResourceLinks, uint64, error) {
	return GetResourceLinks(ctx, coapConn, href, message.Option{
		ID:    message.URIQuery,
		Value: []byte(uri.InterfaceQueryKeyPrefix + interfaces.OC_IF_LL),
	})
}

// GetEndpointsFromResourceType retrieves the endpoints associated with a specific resource type.
func GetEndpointsFromResourceType(ctx context.Context, coapConn ClientConn, resourceType string) ([]string, error) {
	links, _, err := GetResourceLinks(ctx, coapConn, resources.ResourceURI, message.Option{
		ID:    message.URIQuery,
		Value: []byte(uri.InterfaceQueryKeyPrefix + interfaces.OC_IF_LL),
	}, message.Option{
		ID:    message.URIQuery,
		Value: []byte(uri.ResourceTypeQueryKeyPrefix + resourceType),
	})
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, fmt.Errorf("no local endpoints found")
	}
	endpoints := make([]string, 0, 8)
	for _, ep := range links[0].Endpoints {
		endpoints = append(endpoints, ep.URI)
	}
	return endpoints, nil
}

// GetEndpointsFromDeviceResource retrieves the endpoints from the device resource.
func GetEndpointsFromDeviceResource(ctx context.Context, coapConn ClientConn) ([]string, error) {
	return GetEndpointsFromResourceType(ctx, coapConn, device.ResourceType)
}
