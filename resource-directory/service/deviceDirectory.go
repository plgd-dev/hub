package service

import (
	"context"
	"fmt"

	pbDD "github.com/go-ocf/cloud/resource-directory/pb/device-directory"
	"github.com/go-ocf/sdk/schema/cloud"

	coap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/codec/json"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/strings"
	"google.golang.org/grpc/codes"

	"github.com/go-ocf/cloud/resource-aggregate/cqrs"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
)

// hasMatchingStatus returns true for matching a device state.
// An empty status_filter matches all device states.
func hasMatchingStatus(isOnline bool, status_filter []pbDD.Status) bool {
	if len(status_filter) == 0 {
		return true
	}
	for _, f := range status_filter {
		switch f {
		case pbDD.Status_ONLINE:
			if isOnline {
				return true
			}
		case pbDD.Status_OFFLINE:
			if !isOnline {
				return true
			}
		}
	}
	return false
}

type DeviceDirectory struct {
	projection    *Projection
	userDeviceIds strings.Set
}

// NewDeviceDirectory creates new device directory.
func NewDeviceDirectory(projection *Projection, deviceIds []string) *DeviceDirectory {
	mapDeviceIds := make(strings.Set)
	mapDeviceIds.Add(deviceIds...)

	return &DeviceDirectory{projection: projection, userDeviceIds: mapDeviceIds}
}

func decodeContent(content *pbRA.Content, v interface{}) error {
	if content == nil {
		return fmt.Errorf("cannot parse empty content")
	}

	var decoder func([]byte, interface{}) error

	switch content.ContentType {
	case coap.AppCBOR.String(), coap.AppOcfCbor.String():
		decoder = cbor.Decode
	case coap.AppJSON.String():
		decoder = json.Decode
	default:
		return fmt.Errorf("unsupported content type: %v", content.ContentType)
	}

	return decoder(content.Data, v)
}

type Device struct {
	pbDD.Device
	cloudStateUpdated bool
}

func updateDevice(dev Device, resource *resourceCtx) (Device, error) {
	cloudResourceTypes := make(strings.Set)
	cloudResourceTypes.Add(cloud.StatusResourceTypes...)

	switch {
	case resource.snapshot.Resource.Href == "/oic/d":
		var devContent pbDD.Resource
		err := decodeContent(resource.snapshot.GetLatestResourceChange().GetContent(), &devContent)
		if err != nil {
			return dev, err
		}
		dev.Resource = &devContent
		dev.Id = resource.snapshot.GroupId()
	case cloudResourceTypes.HasOneOf(resource.snapshot.Resource.ResourceTypes...):
		var cloudStatus cloud.Status
		err := decodeContent(resource.snapshot.GetLatestResourceChange().GetContent(), &cloudStatus)
		if err != nil {
			return dev, err
		}
		dev.IsOnline = cloudStatus.Online
		dev.cloudStateUpdated = true
	}
	return dev, nil
}

func filterDevicesByUserFilters(resourceValues map[string]map[string]*resourceCtx, req *pbDD.GetDevicesRequest) ([]Device, error) {
	devices := make([]Device, 0, len(resourceValues))
	typeFilter := make(strings.Set)
	typeFilter.Add(req.TypeFilter...)
	for deviceID, resources := range resourceValues {
		var device Device
		for _, resource := range resources {
			dev, err := updateDevice(device, resource)
			if err != nil {
				return nil, fmt.Errorf("cannot process device resources: %w", err)
			}
			device = dev
		}
		if device.Resource == nil {
			device.Id = deviceID
		}
		if !hasMatchingType(device.GetResource().GetResourceTypes(), typeFilter) {
			continue
		}
		if !device.cloudStateUpdated {
			continue
		}

		if hasMatchingStatus(device.IsOnline, req.StatusFilter) {
			devices = append(devices, device)
		}
	}
	return devices, nil
}

// GetDevices provides list state of devices.
func (dd *DeviceDirectory) GetDevices(ctx context.Context, req *pbDD.GetDevicesRequest, responseHandler func(*pbDD.Device) error) (statusCode codes.Code, err error) {
	deviceIds := filterDevices(dd.userDeviceIds, req.DeviceIdsFilter)
	if len(deviceIds) == 0 {
		err = fmt.Errorf("not found")
		statusCode = codes.NotFound
		return
	}

	resourceIdsFilter := make(strings.Set)
	for deviceID := range deviceIds {
		resourceIdsFilter.Add(cqrs.MakeResourceId(deviceID, "/oic/d"))
		resourceIdsFilter.Add(cqrs.MakeResourceId(deviceID, cloud.StatusHref))
	}

	resourceValues, err := dd.projection.GetResourceCtxs(ctx, resourceIdsFilter, nil, deviceIds)
	if err != nil {
		err = fmt.Errorf("cannot get resource links by device ids: %w", err)
		statusCode = codes.Internal
		return
	}

	devices, err := filterDevicesByUserFilters(resourceValues, req)
	if err != nil {
		statusCode = codes.Internal
		return
	}

	if len(devices) == 0 {
		err = fmt.Errorf("not found")
		statusCode = codes.NotFound
		return
	}

	log.Debugf("DeviceDirectory.GetDevices send devices %+v", devices)

	for _, device := range devices {
		dev := device
		if err = responseHandler(&dev.Device); err != nil {
			err = fmt.Errorf("cannot retrieve resources values: %w", err)
			statusCode = codes.Canceled
			return
		}
	}

	statusCode = codes.OK
	return
}
