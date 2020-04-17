package service

import (
	"fmt"

	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/sdk/schema"

	"github.com/go-ocf/cloud/grpc-gateway/pb"

	"github.com/go-ocf/sdk/schema/cloud"

	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/codec/json"
	"github.com/go-ocf/kit/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-ocf/cloud/resource-aggregate/cqrs"
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

func (d Device) ToProto() pb.Device {
	var r pb.Device
	if d.Resource != nil {
		r = pb.SchemaDevice(*d.Resource).ToProto()
	}
	r.IsOnline = d.IsOnline
	r.Id = d.ID
	return r
}

func updateDevice(dev Device, resource *resourceCtx) (Device, error) {
	cloudResourceTypes := make(strings.Set)
	cloudResourceTypes.Add(cloud.StatusResourceTypes...)

	switch {
	case resource.resource.GetHref() == "/oic/d":
		var devContent schema.Device
		err := decodeContent(resource.content.GetContent(), &devContent)
		if err != nil {
			return dev, err
		}
		dev.Resource = &devContent
		dev.ID = devContent.ID
	case cloudResourceTypes.HasOneOf(resource.resource.GetResourceTypes()...):
		var cloudStatus cloud.Status
		err := decodeContent(resource.content.GetContent(), &cloudStatus)
		if err != nil {
			return dev, err
		}
		dev.IsOnline = cloudStatus.Online
		dev.cloudStateUpdated = true
	}
	return dev, nil
}

func filterDevicesByStatus(resourceValues map[string]map[string]*resourceCtx, req *pb.GetDevicesRequest) ([]Device, error) {
	devices := make([]Device, 0, len(resourceValues))
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
			device.ID = deviceID
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

	typeFilter := make(strings.Set)
	typeFilter.Add(req.TypeFilter...)
	resourceIdsFilter := make(strings.Set)
	for deviceID := range deviceIds {
		resourceIdsFilter.Add(cqrs.MakeResourceId(deviceID, "/oic/d"))
		resourceIdsFilter.Add(cqrs.MakeResourceId(deviceID, cloud.StatusHref))
	}

	resourceValues, err := dd.projection.GetResourceCtxs(srv.Context(), resourceIdsFilter, typeFilter, deviceIds)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get resource links by device ids: %v", err)
	}

	devices, err := filterDevicesByStatus(resourceValues, req)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot filter devices by status: %v", err)
	}

	if len(devices) == 0 {
		return status.Errorf(codes.NotFound, "not found")
	}

	for _, device := range devices {
		dev := device.ToProto()
		err := srv.Send(&dev)
		if err != nil {
			return status.Errorf(codes.Canceled, "cannot send device: %v", err)
		}
	}

	return nil
}
