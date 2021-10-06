package client

import (
	"context"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/events"
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

	err = c.GetResourceLinksViaCallback(ctx, []string{deviceID}, nil, func(v *events.ResourceLinksPublished) {
		deviceDetails.Resources = v.GetResources()
	})
	if err != nil {
		return nil, err
	}

	return deviceDetails, nil
}
