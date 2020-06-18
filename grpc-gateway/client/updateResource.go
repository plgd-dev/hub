package client

import (
	"context"
	"fmt"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	codecOcf "github.com/go-ocf/kit/codec/ocf"
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
		codec: codecOcf.VNDOCFCBORCodec{},
	}
	for _, o := range opts {
		cfg = o.applyOnUpdate(cfg)
	}

	data, err := cfg.codec.Encode(request)
	if err != nil {
		return err
	}
	r := pb.UpdateResourceValuesRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		ResourceInterface: cfg.resourceInterface,
		Content: &pb.Content{
			Data:        data,
			ContentType: cfg.codec.ContentFormat().String(),
		},
	}

	resp, err := c.gateway.UpdateResourcesValues(ctx, &r)
	if err != nil {
		return fmt.Errorf("cannot update resource /%v/%v: %w", deviceID, href, err)
	}

	return DecodeContentWithCodec(cfg.codec, resp.GetContent().GetContentType(), resp.GetContent().GetData(), response)
}
