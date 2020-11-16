package client

import (
	"context"

	kitNetCoap "github.com/plgd-dev/kit/net/coap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
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
		codec: CloudCodec{},
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
	codec kitNetCoap.Codec,
	response interface{}) error {
	var resp *pb.ResourceValue
	err := c.RetrieveResourcesByResourceIDs(ctx, MakeResourceIDCallback(deviceID, href, func(v pb.ResourceValue) {
		resp = &v
	}))
	if err != nil {
		return err
	}
	if resp == nil {
		return status.Errorf(codes.NotFound, "not found")
	}

	return DecodeContentWithCodec(codec, resp.GetContent().GetContentType(), resp.GetContent().GetData(), response)
}
