package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

// DeleteResource invokes DELETE command within the resource aggregate, which transparently forwards the request to the device.
func (c *Client) DeleteResource(
	ctx context.Context,
	deviceID string,
	href string,
	response interface{},
	opts ...DeleteOption,
) error {
	cfg := deleteOptions{
		codec: GeneralMessageCodec{},
	}
	for _, o := range opts {
		cfg = o.applyOnDelete(cfg)
	}

	r := pb.DeleteResourceRequest{
		ResourceId:        commands.NewResourceID(deviceID, href),
		ResourceInterface: cfg.resourceInterface,
	}

	resp, err := c.gateway.DeleteResource(ctx, &r)
	if err != nil {
		return fmt.Errorf("cannot delete resource /%v%v: %w", deviceID, href, err)
	}

	content := resp.GetData().GetContent()
	return DecodeContentWithCodec(cfg.codec, content.GetContentType(), content.GetData(), response)
}
