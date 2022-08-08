package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetResourceWithCodec retrieves content of a resource from the client.
func (c *Client) GetResource(
	ctx context.Context,
	deviceID string,
	href string,
	response interface{},
	opts ...GetOption,
) error {
	cfg := getOptions{
		codec: GeneralMessageCodec{},
	}
	for _, o := range opts {
		cfg = o.applyOnGet(cfg)
	}
	if cfg.resourceInterface != "" || cfg.skipShadow {
		return c.getResourceFromDevice(ctx, deviceID, href, cfg.resourceInterface, cfg.codec, response)
	}
	return c.getResource(ctx, deviceID, href, cfg.codec, response)
}

// GetResource retrieves content of a resource from the client.
func (c *Client) getResource(
	ctx context.Context,
	deviceID string,
	href string,
	codec Codec,
	response interface{},
) error {
	var resp *pb.Resource
	err := c.GetResourcesByResourceIDs(ctx, MakeResourceIDCallback(deviceID, href, func(v *pb.Resource) {
		resp = v
	}))
	if err != nil {
		return err
	}
	if resp == nil {
		iter := c.GetResourceLinksIterator(ctx, []string{deviceID})
		defer iter.Close()
		var v events.ResourceLinksPublished
		ok := iter.Next(&v)
		if ok && v.GetDeviceId() == deviceID {
			for _, r := range v.GetResources() {
				if r.GetHref() == href {
					return status.Errorf(codes.Unavailable, "content of resource /%v%v is not stored", deviceID, href)
				}
			}
		}
		return status.Errorf(codes.NotFound, "not found")
	}

	if resp.GetData().GetStatus() == commands.Status_UNKNOWN {
		return status.Errorf(codes.Unavailable, "content of resource /%v%v is not stored", deviceID, href)
	}
	content, err := commands.EventContentToContent(resp.GetData())
	if err != nil {
		return fmt.Errorf("cannot retrieve resource /%v%v: %w", deviceID, href, err)
	}
	return DecodeContentWithCodec(codec, content.GetContentType(), content.GetData(), response)
}
