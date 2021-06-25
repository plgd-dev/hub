package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
)

// UpdateResource updates content in OCF-CBOR format.
func (c *Client) UpdateResource(
	ctx context.Context,
	deviceID string,
	href string,
	request interface{},
	response interface{},
	opts ...UpdateOption,
) error {
	cfg := updateOptions{
		codec: GeneralMessageCodec{},
	}
	for _, o := range opts {
		cfg = o.applyOnUpdate(cfg)
	}

	data, err := cfg.codec.Encode(request)
	if err != nil {
		return err
	}
	r := pb.UpdateResourceRequest{
		ResourceId:        commands.NewResourceID(deviceID, href),
		ResourceInterface: cfg.resourceInterface,
		Content: &pb.Content{
			Data:        data,
			ContentType: cfg.codec.ContentFormat().String(),
		},
	}

	resp, err := c.gateway.UpdateResource(ctx, &r)
	if err != nil {
		return fmt.Errorf("cannot update resource /%v/%v: %w", deviceID, href, err)
	}

	content, err := commands.EventContentToContent(resp)
	if err != nil {
		return fmt.Errorf("cannot update resource /%v/%v: %w", deviceID, href, err)
	}

	return DecodeContentWithCodec(cfg.codec, content.GetContentType(), content.GetData(), response)
}
