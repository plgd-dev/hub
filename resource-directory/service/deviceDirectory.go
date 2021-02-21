package service

import (
	"fmt"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/sdk/schema"

	deviceStatus "github.com/plgd-dev/cloud/coap-gateway/schema/device/status"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"

	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/codec/json"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/strings"
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
	ID                string
	Resource          *schema.Device
	IsOnline          bool
	cloudStateUpdated bool
}

func (d Device) ToProto() *pb.Device {
	r := pb.SchemaDeviceToProto(d.Resource)
	if r == nil {
		r = &pb.Device{
			Id: d.ID,
		}
	}
	r.IsOnline = d.IsOnline
	return r
}

func updateDevice(dev *Device, resource *resourceProjection, resourceLink *commands.Resource) error {
	cloudResourceTypes := make(strings.Set)
	cloudResourceTypes.Add(deviceStatus.ResourceTypes...)
	switch {
	case resource.resourceId.GetHref() == "/oic/d":
		var devContent schema.Device
		err := decodeContent(resource.content.GetContent(), &devContent)
		if err != nil {
			return err
		}
		dev.ID = devContent.ID
		dev.Resource = &devContent
		dev.Resource.ResourceTypes = resourceLink.GetResourceTypes()
		dev.Resource.Interfaces = resourceLink.GetInterfaces()
	case cloudResourceTypes.HasOneOf(resourceLink.GetResourceTypes()...):
		var cloudStatus deviceStatus.Status
		err := decodeContent(resource.content.GetContent(), &cloudStatus)
		if err != nil {
			return err
		}
		dev.IsOnline = cloudStatus.IsOnline()
		dev.cloudStateUpdated = true
	}
	return nil
}

func filterDevicesByUserFilters(resourceProjections map[string]map[string]*resourceProjection, resourceLinks map[string]map[string]*commands.Resource, req *pb.GetDevicesRequest) ([]Device, error) {
	devices := make([]Device, 0, len(resourceProjections))
	typeFilter := make(strings.Set)
	typeFilter.Add(req.TypeFilter...)
	for deviceID, resources := range resourceProjections {
		var device Device
		var err error
		for _, resource := range resources {
			err = updateDevice(&device, resource, resourceLinks[resource.resourceId.GetDeviceId()][resource.resourceId.GetHref()])
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
		if !device.cloudStateUpdated {
			continue
		}
		if hasMatchingStatus(device.IsOnline, req.StatusFilter) {
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
	deviceIDs := filterDevices(dd.userDeviceIds, req.DeviceIdsFilter)
	if len(deviceIDs) == 0 {
		return status.Errorf(codes.NotFound, "not found")
	}

	resourceIdsFilter := make([]*commands.ResourceId, 0, 64)
	for deviceID := range deviceIDs {
		resourceIdsFilter = append(resourceIdsFilter, commands.MakeResourceID(deviceID, "/oic/d"), commands.MakeResourceID(deviceID, commands.StatusHref))
	}

	resourceProjections, err := dd.projection.GetResourceProjections(srv.Context(), resourceIdsFilter, nil)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get resources by device ids: %v", err)
	}

	resourceLinks, err := dd.projection.GetResourceLinks(srv.Context(), deviceIDs, nil)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get resource links by device ids: %v", err)
	}

	devices, err := filterDevicesByUserFilters(resourceProjections, resourceLinks, req)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot filter devices by status: %v", err)
	}

	if len(devices) == 0 {
		return status.Errorf(codes.NotFound, "not found")
	}

	for _, device := range devices {
		err := srv.Send(device.ToProto())
		if err != nil {
			return status.Errorf(codes.Canceled, "cannot send device: %v", err)
		}
	}

	return nil
}
