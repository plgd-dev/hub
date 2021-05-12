package client

import (
	"context"

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
	r := pbGW.RetrieveResourceFromDeviceRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		ResourceInterface: resourceInterface,
	}
	resp, err := c.gateway.RetrieveResourceFromDevice(ctx, &r)
	if err != nil {
		return err
	}

	return DecodeContentWithCodec(codec, resp.GetContent().GetContentType(), resp.GetContent().GetData(), response)
}
