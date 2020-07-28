package service

import (
	"fmt"

	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/sdk/schema"

	"github.com/go-ocf/cloud/grpc-gateway/pb"

	"github.com/go-ocf/sdk/schema/cloud"

	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/codec/json"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
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

func decodeContent(content *pbRA.Content, v interface{}) error {
	if content == nil {
		return fmt.Errorf("cannot parse empty content")
	}

	var decoder func([]byte, interface{}) error

	switch content.ContentType {
	case message.AppCBOR.String(), message.AppOcfCbor.String():
		decoder = cbor.Decode
	case message.AppJSON.String():
		decoder = json.Decode
	default:
		return fmt.Errorf("unsupported content type: %v", content.ContentType)
	}

	return decoder(content.Data, v)
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

func updateDevice(dev *Device, resource *resourceCtx) error {
	cloudResourceTypes := make(strings.Set)
	cloudResourceTypes.Add(cloud.StatusResourceTypes...)
	switch {
	case resource.resource.GetHref() == "/oic/d":
		var devContent schema.Device
		err := decodeContent(resource.content.GetContent(), &devContent)
		if err != nil {
			return err
		}
		dev.Resource = &devContent
		if len(dev.Resource.ResourceTypes) == 0 {
			dev.Resource.ResourceTypes = resource.resource.GetResourceTypes()
		}
		if len(dev.Resource.Interfaces) == 0 {
			dev.Resource.Interfaces = resource.resource.GetInterfaces()
		}
		dev.ID = devContent.ID
	case cloudResourceTypes.HasOneOf(resource.resource.GetResourceTypes()...):
		var cloudStatus cloud.Status
		err := decodeContent(resource.content.GetContent(), &cloudStatus)
		if err != nil {
			return err
		}
		dev.IsOnline = cloudStatus.Online
		dev.cloudStateUpdated = true
	}
	return nil
}

func filterDevicesByUserFilters(resourceValues map[string]map[string]*resourceCtx, req *pb.GetDevicesRequest) ([]Device, error) {
	devices := make([]Device, 0, len(resourceValues))
	typeFilter := make(strings.Set)
	typeFilter.Add(req.TypeFilter...)
	for deviceID, resources := range resourceValues {
		var device Device
		var err error
		for _, resource := range resources {
			err = updateDevice(&device, resource)
			if err != nil {
				break
			}
		}
		if err != nil {
			log.Debugf("filterDevicesByUserFilters: cannot process device(%v) resources: %w", deviceID, err)
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
	deviceIds := filterDevices(dd.userDeviceIds, req.DeviceIdsFilter)
	if len(deviceIds) == 0 {
		return status.Errorf(codes.NotFound, "not found")
	}

	resourceIdsFilter := make([]*pb.ResourceId, 0, 64)
	for deviceID := range deviceIds {
		resourceIdsFilter = append(resourceIdsFilter, &pb.ResourceId{DeviceId: deviceID, Href: "/oic/d"}, &pb.ResourceId{DeviceId: deviceID, Href: cloud.StatusHref})
	}

	resourceValues, err := dd.projection.GetResourceCtxs(srv.Context(), resourceIdsFilter, nil, nil)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get resource links by device ids: %v", err)
	}

	devices, err := filterDevicesByUserFilters(resourceValues, req)
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
