package client

import (
	"context"

	pbGW "github.com/go-ocf/cloud/grpc-gateway/pb"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
)

func (c *Client) getResourceFromDevice(
	ctx context.Context,
	deviceID string,
	href string,
	resourceInterface string,
	codec kitNetCoap.Codec,
	response interface{},
) error {
	r := pbGW.RetrieveResourceFromDeviceRequest{
		ResourceId: &pbGW.ResourceId{
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
