package client

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetDevice retrieves device details from the client.
func (c *Client) GetDevice(
	ctx context.Context,
	deviceID string,
) (*DeviceDetails, error) {
	var deviceDetails *DeviceDetails
	err := c.GetDevicesViaCallback(ctx, []string{deviceID}, nil, func(v *pb.Device) {
		deviceDetails = &DeviceDetails{
			ID:     v.GetId(),
			Device: v,
		}
	})
	if err != nil {
		return nil, err
	}
	if deviceDetails == nil {
		return nil, status.Errorf(codes.NotFound, "not found")
	}

	err = c.GetResourceLinksViaCallback(ctx, []string{deviceID}, nil, func(v *pb.ResourceLink) {
		deviceDetails.Resources = append(deviceDetails.Resources, v)
	})
	if err != nil {
		return nil, err
	}

	return deviceDetails, nil
}
