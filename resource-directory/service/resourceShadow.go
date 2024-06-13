package service

import (
	"bytes"
	"context"
	"time"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
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

func (rd *ResourceTwin) convertToResourceIDs(resourceIDsFilter []*pb.ResourceIdFilter, deviceIdFilter []string) []*pb.ResourceIdFilter {
	internalResourceIDsFilter := make([]*pb.ResourceIdFilter, 0, len(resourceIDsFilter)+len(deviceIdFilter))
	for _, r := range resourceIDsFilter {
		if rd.userDeviceIds.HasOneOf(r.GetResourceId().GetDeviceId()) {
			internalResourceIDsFilter = append(internalResourceIDsFilter, r)
		}
	}
	for _, deviceID := range deviceIdFilter {
		if rd.userDeviceIds.HasOneOf(deviceID) {
			internalResourceIDsFilter = append(internalResourceIDsFilter, &pb.ResourceIdFilter{
				ResourceId: commands.NewResourceID(deviceID, ""),
			})
		}
	}
	if len(internalResourceIDsFilter) == 0 {
		if len(resourceIDsFilter) > 0 || len(deviceIdFilter) > 0 {
			return nil
		}
		internalResourceIDsFilter = make([]*pb.ResourceIdFilter, 0, len(rd.userDeviceIds))
		for userDeviceID := range rd.userDeviceIds {
			internalResourceIDsFilter = append(internalResourceIDsFilter, &pb.ResourceIdFilter{
				ResourceId: commands.NewResourceID(userDeviceID, ""),
			})
		}
	}
	return internalResourceIDsFilter
}

func (rd *ResourceTwin) filterResources(ctx context.Context, resourceIDsFilter []*commands.ResourceId, typeFilter []string, includeHiddenResources bool, toReloadDevices strings.Set, onResource func(*Resource) error) error {
	mapTypeFilter := make(strings.Set)
	mapTypeFilter.Add(typeFilter...)
	return rd.projection.LoadResources(ctx, resourceIDsFilter, mapTypeFilter, includeHiddenResources, toReloadDevices, onResource)
}

func resourceIdFilterToSimple(r []*pb.ResourceIdFilter) []*commands.ResourceId {
	if len(r) == 0 {
		return nil
	}
	res := make([]*commands.ResourceId, 0, len(r))
	for _, v := range r {
		res = append(res, v.GetResourceId())
	}
	return res
}

func updateContentForResponseForETag(v *pb.ResourceIdFilter, val *pb.Resource) bool {
	for _, etag := range v.GetEtag() {
		if !bytes.Equal(etag, val.GetData().GetEtag()) {
			continue
		}
		rc := val.GetData()
		data := &events.ResourceChanged{}
		data.CopyData(rc)
		data.Status = commands.Status_NOT_MODIFIED
		data.Content = &commands.Content{
			CoapContentFormat: int32(-1),
		}
		val.Data = data
		return true
	}
	return false
}

func updateContentIfETagMatched(resourceIDsFilter []*pb.ResourceIdFilter, val *pb.Resource) {
	rc := val.GetData()
	for _, v := range resourceIDsFilter {
		if len(v.GetEtag()) == 0 {
			continue
		}
		if !v.GetResourceId().Equal(rc.GetResourceId()) {
			continue
		}
		if updateContentForResponseForETag(v, val) {
			return
		}
	}
}

func (rd *ResourceTwin) getResources(resourceIDsFilter []*pb.ResourceIdFilter, typeFilter []string, srv pb.GrpcGateway_GetResourcesServer, toReloadDevices strings.Set) error {
	return rd.filterResources(srv.Context(), resourceIdFilterToSimple(resourceIDsFilter), typeFilter, false, toReloadDevices, func(resource *Resource) error {
		val := toResourceValue(resource)
		updateContentIfETagMatched(resourceIDsFilter, val)
		err := srv.Send(val)
		if err != nil {
			return status.Errorf(codes.Canceled, "cannot send resource value %v: %v", val, err)
		}
		return nil
	})
}

func (rd *ResourceTwin) GetResources(req *pb.GetResourcesRequest, srv pb.GrpcGateway_GetResourcesServer) error {
	// for backward compatibility and http api
	req.ResourceIdFilter = append(req.ResourceIdFilter, req.ConvertHTTPResourceIDFilter()...)

	resourceIDsFilter := rd.convertToResourceIDs(req.GetResourceIdFilter(), req.GetDeviceIdFilter())
	toReloadDevices := make(strings.Set)
	err := rd.getResources(resourceIDsFilter, req.GetTypeFilter(), srv, toReloadDevices)
	if err != nil {
		return err
	}
	if len(toReloadDevices) > 0 {
		rd.projection.ReloadDevices(srv.Context(), toReloadDevices)
		newResourceIDsFilter := make([]*pb.ResourceIdFilter, 0, len(resourceIDsFilter))
		for i := range resourceIDsFilter {
			if toReloadDevices.HasOneOf(resourceIDsFilter[i].GetResourceId().GetDeviceId()) {
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

func (rd *ResourceTwin) sendPendingCommands(srv pb.GrpcGateway_GetPendingCommandsServer, resourceIDsFilter []*pb.ResourceIdFilter, typeFilter []string, filterCmds subscription.FilterBitmask, includeHiddenResources bool, now time.Time, toReloadDevices strings.Set) error {
	return rd.filterResources(srv.Context(), resourceIdFilterToSimple(resourceIDsFilter), typeFilter, includeHiddenResources, toReloadDevices, func(resource *Resource) error {
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
	// for backward compatibility and http api
	req.ResourceIdFilter = append(req.ResourceIdFilter, req.ConvertHTTPResourceIDFilter()...)

	err := rd.getDeviceMetadataUpdatePendingCommands(req, srv, now, filterCmds)
	if err != nil {
		return err
	}

	resourceIDsFilter := rd.convertToResourceIDs(req.GetResourceIdFilter(), req.GetDeviceIdFilter())
	toReloadDevices := make(strings.Set)
	err = rd.sendPendingCommands(srv, resourceIDsFilter, req.GetTypeFilter(), filterCmds, req.GetIncludeHiddenResources(), now, toReloadDevices)
	if err != nil {
		return err
	}
	if len(toReloadDevices) > 0 {
		rd.projection.ReloadDevices(srv.Context(), toReloadDevices)
		newResourceIDsFilter := make([]*pb.ResourceIdFilter, 0, len(resourceIDsFilter))
		for i := range resourceIDsFilter {
			if toReloadDevices.HasOneOf(resourceIDsFilter[i].GetResourceId().GetDeviceId()) {
				newResourceIDsFilter = append(newResourceIDsFilter, resourceIDsFilter[i])
			}
		}
		return rd.sendPendingCommands(srv, newResourceIDsFilter, req.GetTypeFilter(), filterCmds, req.GetIncludeHiddenResources(), now, nil)
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
		if len(typeFilter) > 0 && !typeFilter.HasOneOf(res.GetResourceTypes()...) {
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
	deviceIDs := filterDevices(rd.userDeviceIds, req.GetDeviceIdFilter())
	typeFilter := make(strings.Set)
	typeFilter.Add(req.GetTypeFilter()...)
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
