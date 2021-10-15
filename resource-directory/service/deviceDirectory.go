package service

import (
	"fmt"

	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/go-coap/v2/message"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"

	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/plgd-dev/kit/v2/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// hasMatchingStatus returns true for matching a device state.
// An empty statusFilter matches all device states.
func hasMatchingStatus(isOnline bool, statusFilter []pb.GetDevicesRequest_Status) bool {
	if len(statusFilter) == 0 {
		return true
	}
	for _, f := range statusFilter {
		switch f {
		case pb.GetDevicesRequest_ONLINE:
			if isOnline {
				return true
			}
		case pb.GetDevicesRequest_OFFLINE:
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

func decodeContent(content *commands.Content, v interface{}) error {
	if content == nil {
		return fmt.Errorf("cannot parse empty content")
	}

	var decoder func([]byte, interface{}) error

	switch content.GetContentType() {
	case message.AppCBOR.String(), message.AppOcfCbor.String():
		decoder = cbor.Decode
	case message.AppJSON.String():
		decoder = json.Decode
	default:
		return fmt.Errorf("unsupported content type: %v", content.GetContentType())
	}

	return decoder(content.GetData(), v)
}

type Device struct {
	ID       string
	Resource *device.Device
	Metadata *pb.Device_Metadata
}

func (d Device) ToProto() *pb.Device {
	r := pb.SchemaDeviceToProto(d.Resource)
	if r == nil {
		r = &pb.Device{
			Id: d.ID,
		}
	}
	r.Metadata = d.Metadata
	return r
}

func updateDevice(dev *Device, resource *Resource) error {
	switch {
	case resource.Resource.GetHref() == device.ResourceURI:
		var devContent device.Device
		err := decodeContent(resource.GetContent(), &devContent)
		if err != nil {
			return err
		}
		dev.ID = devContent.ID
		dev.Resource = &devContent
		dev.Resource.ResourceTypes = resource.Resource.GetResourceTypes()
		dev.Resource.Interfaces = resource.Resource.GetInterfaces()
	}
	return nil
}

func filterDevicesByUserFilters(resources map[string]map[string]*Resource, devicesMetadata map[string]*events.DeviceMetadataSnapshotTaken, req *pb.GetDevicesRequest) ([]Device, error) {
	devices := make([]Device, 0, len(resources))
	typeFilter := make(strings.Set)
	typeFilter.Add(req.TypeFilter...)
	for deviceID, resources := range resources {
		var device Device
		var err error
		for _, resource := range resources {
			err = updateDevice(&device, resource)
			if err != nil {
				break
			}
		}
		if err != nil {
			log.Debugf("filterDevicesByUserFilters: cannot process device(%v) resources: %v", deviceID, err)
			continue
		}
		var resourceTypes []string
		if device.Resource == nil {
			device.ID = deviceID
		} else {
			resourceTypes = device.Resource.ResourceTypes
		}
		if !hasMatchingType(resourceTypes, typeFilter) {
			continue
		}

		deviceMetadata, ok := devicesMetadata[deviceID]
		if !ok {
			continue
		}
		if hasMatchingStatus(deviceMetadata.GetDeviceMetadataUpdated().GetStatus().IsOnline(), req.StatusFilter) {
			device.Metadata = &pb.Device_Metadata{
				Status:                deviceMetadata.GetDeviceMetadataUpdated().GetStatus(),
				ShadowSynchronization: deviceMetadata.GetDeviceMetadataUpdated().GetShadowSynchronization(),
			}

			devices = append(devices, device)
		}
	}

	return devices, nil
}

// filterDevices returns filtered device ids that match filter.
// An empty deviceIDsFilter matches all device ids.
func filterDevices(deviceIds strings.Set, deviceIDsFilter []string) strings.Set {
	if len(deviceIDsFilter) == 0 {
		return deviceIds
	}
	result := make(strings.Set)
	for _, deviceID := range deviceIDsFilter {
		if deviceIds.HasOneOf(deviceID) {
			result.Add(deviceID)
		}
	}
	return result
}

// GetDevices provides list state of devices.
func (dd *DeviceDirectory) GetDevices(req *pb.GetDevicesRequest, srv pb.GrpcGateway_GetDevicesServer) (err error) {
	deviceIDs := filterDevices(dd.userDeviceIds, req.DeviceIdFilter)
	if len(deviceIDs) == 0 {
		log.Debug("DeviceDirectory.GetDevices.filterDevices returns empty deviceIDs")
		return nil
	}

	resourceIdFilter := make([]*commands.ResourceId, 0, 64)
	for deviceID := range deviceIDs {
		resourceIdFilter = append(resourceIdFilter, commands.NewResourceID(deviceID, device.ResourceURI), commands.NewResourceID(deviceID, commands.StatusHref))
	}

	resources, err := dd.projection.GetResourcesWithLinks(srv.Context(), resourceIdFilter, nil)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get resources by device ids: %v", err)
	}

	devicesMetadata, err := dd.projection.GetDevicesMetadata(srv.Context(), deviceIDs)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get devices metadata device ids: %v", err)
	}

	devices, err := filterDevicesByUserFilters(resources, devicesMetadata, req)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot filter devices by status: %v", err)
	}

	if len(devices) == 0 {
		log.Debug("DeviceDirectory.GetDevices.filterDevicesByUserFilters returns empty devices")
		return nil
	}

	for _, device := range devices {
		err := srv.Send(device.ToProto())
		if err != nil {
			return status.Errorf(codes.Canceled, "cannot send device: %v", err)
		}
	}

	return nil
}
