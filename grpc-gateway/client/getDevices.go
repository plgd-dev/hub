package client

import (
	"context"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
)

// DeviceDetails describes a device.
type DeviceDetails struct {
	// ID of the device
	ID string
	// Device basic content(oic.wk.d) of /oic/d resource.
	Device pb.Device
	// Resources list of the device resources.
	Resources []*pb.ResourceLink
}

// GetDevices retrieves device details from the client.
func (c *Client) GetDevices(
	ctx context.Context,
	opts ...GetDevicesOption,
) (map[string]DeviceDetails, error) {
	var cfg getDevicesOptions
	for _, o := range opts {
		cfg = o.applyOnGetDevices(cfg)
	}

	devices := make(map[string]DeviceDetails, len(cfg.deviceIDs))
	ids := make([]string, 0, len(cfg.deviceIDs))

	err := c.GetDevicesViaCallback(ctx, cfg.deviceIDs, cfg.resourceTypes, func(v pb.Device) {
		devices[v.GetId()] = DeviceDetails{
			ID:     v.GetId(),
			Device: v,
		}
		ids = append(ids, v.GetId())
	})
	if err != nil {
		return nil, err
	}

	err = c.GetResourceLinksViaCallback(ctx, ids, nil, func(v *pb.ResourceLink) {
		d, ok := devices[v.GetDeviceId()]
		if ok {
			d.Resources = append(d.Resources, v)
			devices[v.GetDeviceId()] = d
		}
	})
	if err != nil {
		return nil, err
	}

	return devices, nil
}
