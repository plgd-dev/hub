package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
)

// CreateResource requests creation of a new resource on a collection resource on a device.
func (c *Client) CreateResource(
	ctx context.Context,
	deviceID string,
	href string,
	request interface{},
	response interface{},
	opts ...CreateOption,
) error {
	cfg := createOptions{
		codec: GeneralMessageCodec{},
	}
	for _, o := range opts {
		cfg = o.applyOnCreate(cfg)
	}

	data, err := cfg.codec.Encode(request)
	if err != nil {
		return err
	}
	r := pb.CreateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, href),
		Content: &pb.Content{
			Data:        data,
			ContentType: cfg.codec.ContentFormat().String(),
		},
	}

	resp, err := c.gateway.CreateResource(ctx, &r)
	if err != nil {
		return fmt.Errorf("cannot create resource /%v/%v: %w", deviceID, href, err)
	}
	content := resp.GetData().GetContent()
	return DecodeContentWithCodec(cfg.codec, content.GetContentType(), content.GetData(), response)
}
