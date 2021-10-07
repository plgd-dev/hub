package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
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
	response interface{}) error {
	var resp *pb.Resource
	err := c.GetResourcesByResourceIDs(ctx, MakeResourceIDCallback(deviceID, href, func(v *pb.Resource) {
		resp = v
	}))
	if err != nil {
		return err
	}
	if resp == nil {
		return status.Errorf(codes.NotFound, "not found")
	}

	switch resp.GetData().GetStatus() {
	case commands.Status_UNKNOWN:
		return status.Errorf(codes.Unavailable, "content of resource /%v%v is not stored", deviceID, href)
	}
	content, err := commands.EventContentToContent(resp.GetData())
	if err != nil {
		return fmt.Errorf("cannot retrieve resource /%v%v: %w", deviceID, href, err)
	}
	return DecodeContentWithCodec(codec, content.GetContentType(), content.GetData(), response)
}
