package client

import (
	"context"
	"fmt"

	pbGW "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
)

func (c *Client) getResourceFromDevice(
	ctx context.Context,
	deviceID string,
	href string,
	resourceInterface string,
	codec Codec,
	response interface{},
) error {
	r := pbGW.GetResourceFromDeviceRequest{
		ResourceId:        commands.NewResourceID(deviceID, href),
		ResourceInterface: resourceInterface,
	}
	resp, err := c.gateway.GetResourceFromDevice(ctx, &r)
	if err != nil {
		return fmt.Errorf("cannot retrieve resource from device /%v%v: %w", deviceID, href, err)
	}

	content := resp.GetData().GetContent()
	return DecodeContentWithCodec(codec, content.GetContentType(), content.GetData(), response)
}
