package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
)

// DeleteResource deletes resource.
func (c *Client) DeleteResource(
	ctx context.Context,
	deviceID string,
	href string,
	response interface{},
	opts ...DeleteOption,
) error {
	cfg := deleteOptions{
		codec: CloudCodec{},
	}
	for _, o := range opts {
		cfg = o.applyOnDelete(cfg)
	}

	r := pb.DeleteResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
	}

	resp, err := c.gateway.DeleteResource(ctx, &r)
	if err != nil {
		return fmt.Errorf("cannot delete resource /%v/%v: %w", deviceID, href, err)
	}

	return DecodeContentWithCodec(cfg.codec, resp.GetContent().GetContentType(), resp.GetContent().GetData(), response)
}
