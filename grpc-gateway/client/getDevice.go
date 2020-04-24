package client

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetDevice retrieves device details from the client.
func (c *Client) GetDevice(
	ctx context.Context,
	deviceID string,
) (DeviceDetails, error) {
	devices, err := c.GetDevices(ctx, WithDeviceIDs(deviceID))
	if err != nil {
		return DeviceDetails{}, err
	}
	if len(devices) == 0 {
		return DeviceDetails{}, status.Errorf(codes.NotFound, "not found")
	}
	return devices[deviceID], nil
}
