package service

import (
	"time"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/kit/v2/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toResourceValue(resource *Resource) *pb.Resource {
	return &pb.Resource{
		Data:  resource.GetResourceChanged(),
		Types: resource.Resource.GetResourceTypes(),
	}
}

type ResourceTwin struct {
	projection    *Projection
	userDeviceIds strings.Set
}

func NewResourceTwin(projection *Projection, deviceIds []string) *ResourceTwin {
	mapDeviceIds := make(strings.Set)
	mapDeviceIds.Add(deviceIds...)

	return &ResourceTwin{projection: projection, userDeviceIds: mapDeviceIds}
}

func (rd *ResourceTwin) convertToResourceIDs(resourceIDsFilter, deviceIdFilter []string) []*commands.ResourceId {
	internalResourceIDsFilter := make([]*commands.ResourceId, 0, len(resourceIDsFilter)+len(deviceIdFilter))
	for _, r := range resourceIDsFilter {
		res := commands.ResourceIdFromString(r)

		if rd.userDeviceIds.HasOneOf(res.GetDeviceId()) {
			internalResourceIDsFilter = append(internalResourceIDsFilter, res)
		}
	}
	for _, deviceID := range deviceIdFilter {
		if rd.userDeviceIds.HasOneOf(deviceID) {
			internalResourceIDsFilter = append(internalResourceIDsFilter, commands.NewResourceID(deviceID, ""))
		}
	}
	if len(internalResourceIDsFilter) == 0 {
		if len(resourceIDsFilter) > 0 || len(deviceIdFilter) > 0 {
			return nil
		}
		internalResourceIDsFilter = make([]*commands.ResourceId, 0, len(rd.userDeviceIds))
		for userDeviceID := range rd.userDeviceIds {
			internalResourceIDsFilter = append(internalResourceIDsFilter, commands.NewResourceID(userDeviceID, ""))
		}
	}
	return internalResourceIDsFilter
}

func (rd *ResourceTwin) filterResources(resourceIDsFilter []*commands.ResourceId, typeFilter []string, toReloadDevices strings.Set, onResource func(*Resource) error) error {
	mapTypeFilter := make(strings.Set)
	mapTypeFilter.Add(typeFilter...)
	return rd.projection.LoadResourcesWithLinks(resourceIDsFilter, mapTypeFilter, toReloadDevices, onResource)
}

func (rd *ResourceTwin) getResources(resourceIDsFilter []*commands.ResourceId, typeFilter []string, srv pb.GrpcGateway_GetResourcesServer, toReloadDevices strings.Set) error {
	return rd.filterResources(resourceIDsFilter, typeFilter, toReloadDevices, func(resource *Resource) error {
		val := toResourceValue(resource)
		err := srv.Send(val)
		if err != nil {
			return status.Errorf(codes.Canceled, "cannot send resource value %v: %v", val, err)
		}
		return nil
	})
}

func (rd *ResourceTwin) GetResources(req *pb.GetResourcesRequest, srv pb.GrpcGateway_GetResourcesServer) error {
	toReloadDevices := make(strings.Set)
	resourceIDsFilter := rd.convertToResourceIDs(req.GetResourceIdFilter(), req.GetDeviceIdFilter())
	err := rd.getResources(resourceIDsFilter, req.GetTypeFilter(), srv, toReloadDevices)
	if err != nil {
		return err
	}
	if len(toReloadDevices) > 0 {
		rd.projection.ReloadDevices(srv.Context(), toReloadDevices)
		newResourceIDsFilter := make([]*commands.ResourceId, 0, len(resourceIDsFilter))
		for i := range resourceIDsFilter {
			if toReloadDevices.HasOneOf(resourceIDsFilter[i].GetDeviceId()) {
				newResourceIDsFilter = append(newResourceIDsFilter, resourceIDsFilter[i])
			}
		}
		return rd.getResources(newResourceIDsFilter, req.GetTypeFilter(), srv, nil)
	}
	return nil
}

func toPendingCommands(resource *Resource, commandFilter subscription.FilterBitmask, now time.Time) []*pb.PendingCommand {
	if resource.projection == nil {
		return nil
	}
	return resource.projection.ToPendingCommands(commandFilter, now)
}

func (rd *ResourceTwin) sendPendingCommands(srv pb.GrpcGateway_GetPendingCommandsServer, resourceIDsFilter []*commands.ResourceId, typeFilter []string, filterCmds subscription.FilterBitmask, now time.Time, toReloadDevices strings.Set) error {
	return rd.filterResources(resourceIDsFilter, typeFilter, toReloadDevices, func(resource *Resource) error {
		for _, pendingCmd := range toPendingCommands(resource, filterCmds, now) {
			err := srv.Send(pendingCmd)
			if err != nil {
				return status.Errorf(codes.Canceled, "cannot send pending command %v: %v", pendingCmd, err)
			}
		}
		return nil
	})
}

func (rd *ResourceTwin) sendDeviceMetadataUpdatePendingCommands(deviceIDs strings.Set, srv pb.GrpcGateway_GetPendingCommandsServer, now time.Time, toReloadDevices strings.Set) error {
	return rd.projection.LoadDevicesMetadata(deviceIDs, toReloadDevices, func(m *deviceMetadataProjection) error {
		for _, pendingCmd := range m.GetDeviceUpdatePendings(now) {
			err := srv.Send(&pb.PendingCommand{
				Command: &pb.PendingCommand_DeviceMetadataUpdatePending{
					DeviceMetadataUpdatePending: pendingCmd,
				},
			})
			if err != nil {
				return status.Errorf(codes.Canceled, "cannot send device metadata update pending command: %v", err)
			}
		}
		return nil
	})
}

func (rd *ResourceTwin) getDeviceMetadataUpdatePendingCommands(req *pb.GetPendingCommandsRequest, srv pb.GrpcGateway_GetPendingCommandsServer, now time.Time, filterCmds subscription.FilterBitmask) error {
	toReloadDevices := make(strings.Set)
	if subscription.IsFilteredBit(filterCmds, subscription.FilterBitmaskDeviceMetadataUpdatePending) &&
		len(req.GetResourceIdFilter()) == 0 && len(req.GetTypeFilter()) == 0 {
		deviceIDs := filterDevices(rd.userDeviceIds, req.GetDeviceIdFilter())
		err := rd.sendDeviceMetadataUpdatePendingCommands(deviceIDs, srv, now, toReloadDevices)
		if err != nil {
			return err
		}
	}
	if len(toReloadDevices) > 0 {
		rd.projection.ReloadDevices(srv.Context(), toReloadDevices)
		return rd.sendDeviceMetadataUpdatePendingCommands(toReloadDevices, srv, now, nil)
	}
	return nil
}

func (rd *ResourceTwin) GetPendingCommands(req *pb.GetPendingCommandsRequest, srv pb.GrpcGateway_GetPendingCommandsServer) error {
	filterCmds := subscription.FilterPendingsCommandsToBitmask(req.GetCommandFilter())
	now := time.Now()

	err := rd.getDeviceMetadataUpdatePendingCommands(req, srv, now, filterCmds)
	if err != nil {
		return err
	}

	toReloadDevices := make(strings.Set)
	resourceIDsFilter := rd.convertToResourceIDs(req.GetResourceIdFilter(), req.GetDeviceIdFilter())
	err = rd.sendPendingCommands(srv, resourceIDsFilter, req.GetTypeFilter(), filterCmds, now, toReloadDevices)
	if err != nil {
		return err
	}
	if len(toReloadDevices) > 0 {
		rd.projection.ReloadDevices(srv.Context(), toReloadDevices)
		newResourceIDsFilter := make([]*commands.ResourceId, 0, len(resourceIDsFilter))
		for i := range resourceIDsFilter {
			if toReloadDevices.HasOneOf(resourceIDsFilter[i].GetDeviceId()) {
				newResourceIDsFilter = append(newResourceIDsFilter, resourceIDsFilter[i])
			}
		}
		return rd.sendPendingCommands(srv, newResourceIDsFilter, req.GetTypeFilter(), filterCmds, now, nil)
	}
	return nil
}

func (rd *ResourceTwin) sendDevicesMetadata(srv pb.GrpcGateway_GetDevicesMetadataServer, deviceIDFilter, typeFilter, toReloadDevices strings.Set) error {
	return rd.projection.LoadResourceLinks(deviceIDFilter, toReloadDevices, func(m *resourceLinksProjection) error {
		res := m.GetResource(device.ResourceURI)
		if res == nil {
			if toReloadDevices != nil {
				toReloadDevices.Add(m.GetDeviceID())
			}
			return nil
		}
		if len(typeFilter) > 0 && !typeFilter.HasOneOf(res.ResourceTypes...) {
			return nil
		}
		return rd.projection.LoadDevicesMetadata(strings.MakeSet(m.GetDeviceID()), toReloadDevices, func(m *deviceMetadataProjection) error {
			data := m.GetDeviceMetadataUpdated()
			if data == nil {
				return nil
			}
			err := srv.Send(data)
			if err != nil {
				return status.Errorf(codes.Canceled, "cannot send devices metadata %v: %v", data, err)
			}
			return nil
		})
	})
}

func (rd *ResourceTwin) GetDevicesMetadata(req *pb.GetDevicesMetadataRequest, srv pb.GrpcGateway_GetDevicesMetadataServer) error {
	deviceIDs := filterDevices(rd.userDeviceIds, req.DeviceIdFilter)
	typeFilter := make(strings.Set)
	typeFilter.Add(req.TypeFilter...)
	toReloadDevices := make(strings.Set)
	err := rd.sendDevicesMetadata(srv, deviceIDs, typeFilter, toReloadDevices)
	if err != nil {
		return err
	}
	if len(toReloadDevices) > 0 {
		rd.projection.ReloadDevices(srv.Context(), toReloadDevices)
		return rd.sendDevicesMetadata(srv, toReloadDevices, typeFilter, nil)
	}
	return nil
}
