package service

import (
	"fmt"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
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
	ID              string
	Content         *device.Device
	ResourceChanged *events.ResourceChanged
	Metadata        *pb.Device_Metadata
	Endpoints       []*commands.EndpointInformation
}

func (d Device) ToProto() *pb.Device {
	r := pb.SchemaDeviceToProto(d.Content)
	if r == nil {
		r = &pb.Device{
			Id: d.ID,
		}
	}
	r.Metadata = d.Metadata
	r.Data = d.ResourceChanged
	r.OwnershipStatus = pb.Device_OWNED
	if len(d.Endpoints) == 0 {
		return r
	}
	r.Endpoints = make([]string, 0, len(d.Endpoints))
	for _, endpoint := range d.Endpoints {
		r.Endpoints = append(r.Endpoints, endpoint.GetEndpoint())
	}
	return r
}

func updateDevice(dev *Device, resource *Resource) error {
	if resource.Resource.GetHref() == device.ResourceURI {
		var devContent device.Device
		err := decodeContent(resource.GetContent(), &devContent)
		if err != nil {
			return err
		}
		dev.ID = devContent.ID
		dev.Content = &devContent
		dev.Content.ResourceTypes = resource.Resource.GetResourceTypes()
		dev.Content.Interfaces = resource.Resource.GetInterfaces()
		dev.Endpoints = resource.Resource.GetEndpointInformations()
		dev.ResourceChanged = resource.GetResourceChanged()
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
	return dd.projection.LoadDevicesMetadata(deviceIDs, toReloadDevices, func(m *deviceMetadataProjection) error {
		deviceMetadataUpdated := m.GetDeviceMetadataUpdated()
		if !hasMatchingStatus(deviceMetadataUpdated.GetConnection().IsOnline(), req.StatusFilter) {
			return nil
		}
		resourceIdFilter := []*commands.ResourceId{commands.NewResourceID(m.GetDeviceID(), device.ResourceURI)}
		return dd.projection.LoadResourcesWithLinks(resourceIdFilter, typeFilter, toReloadDevices, func(resource *Resource) error {
			var device Device
			err = updateDevice(&device, resource)
			if err != nil {
				// device is not valid
				return nil //nolint:nilerr
			}
			device.Metadata = &pb.Device_Metadata{
				Connection:          deviceMetadataUpdated.GetConnection(),
				TwinSynchronization: deviceMetadataUpdated.GetTwinSynchronization(),
				TwinEnabled:         deviceMetadataUpdated.GetTwinEnabled(),
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
