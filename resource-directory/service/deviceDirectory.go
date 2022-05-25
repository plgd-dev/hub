package service

import (
	"fmt"

	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
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

func (dd *DeviceDirectory) sendDevices(deviceIDs strings.Set, req *pb.GetDevicesRequest, srv pb.GrpcGateway_GetDevicesServer, toReloadDevices strings.Set) (err error) {
	typeFilter := make(strings.Set)
	typeFilter.Add(req.TypeFilter...)
	return dd.projection.LoadDevicesMetadata(srv.Context(), deviceIDs, toReloadDevices, func(m *deviceMetadataProjection) error {
		deviceMetadataUpdated := m.GetDeviceMetadataUpdated()
		if !hasMatchingStatus(deviceMetadataUpdated.GetStatus().IsOnline(), req.StatusFilter) {
			return nil
		}
		resourceIdFilter := []*commands.ResourceId{commands.NewResourceID(m.GetDeviceID(), device.ResourceURI)}
		return dd.projection.LoadResourcesWithLinks(srv.Context(), resourceIdFilter, typeFilter, toReloadDevices, func(resource *Resource) error {
			var device Device
			err = updateDevice(&device, resource)
			if err != nil {
				// device is not valid
				return nil
			}
			device.Metadata = &pb.Device_Metadata{
				Status:                deviceMetadataUpdated.GetStatus(),
				ShadowSynchronization: deviceMetadataUpdated.GetShadowSynchronization(),
			}
			err := srv.Send(device.ToProto())
			if err != nil {
				return status.Errorf(codes.Canceled, "cannot send device: %v", err)
			}
			return nil
		})
	})
}

func (dd *DeviceDirectory) GetDevices(req *pb.GetDevicesRequest, srv pb.GrpcGateway_GetDevicesServer) (err error) {
	deviceIDs := filterDevices(dd.userDeviceIds, req.DeviceIdFilter)
	if len(deviceIDs) == 0 {
		log.Debug("DeviceDirectory.GetDevices.filterDevices returns empty deviceIDs")
		return nil
	}

	toReloadDevices := make(strings.Set)
	err = dd.sendDevices(deviceIDs, req, srv, toReloadDevices)
	if err != nil {
		return err
	}

	if len(toReloadDevices) > 0 {
		dd.projection.ReloadDevices(srv.Context(), toReloadDevices)
		return dd.sendDevices(toReloadDevices, req, srv, nil)
	}

	return nil
}
