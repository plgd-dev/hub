package service

import (
	"context"

	kitStrings "github.com/plgd-dev/kit/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

func toResourceValue(resource *Resource, encoder func(ec *commands.Content) (*commands.Content, error)) (*pb.Resource, error) {
	content, err := encoder(resource.projection.content.GetContent())
	if err != nil {
		return nil, err
	}
	data := resource.projection.content.Clone()
	if data != nil {
		data.Content = content
	}

	return &pb.Resource{
		Data:  data,
		Types: resource.Resource.GetResourceTypes(),
	}, nil
}

type ResourceShadow struct {
	projection    *Projection
	userDeviceIds kitStrings.Set
}

func NewResourceShadow(projection *Projection, deviceIds []string) *ResourceShadow {
	mapDeviceIds := make(kitStrings.Set)
	mapDeviceIds.Add(deviceIds...)

	return &ResourceShadow{projection: projection, userDeviceIds: mapDeviceIds}
}

func (rd *ResourceShadow) filterResources(ctx context.Context, resourceIDsFilter, deviceIdsFilter, typeFilter []string) (map[string]map[string]*Resource, error) {
	mapTypeFilter := make(kitStrings.Set)
	mapTypeFilter.Add(typeFilter...)

	internalResourceIDsFilter := make([]*commands.ResourceId, 0, len(resourceIDsFilter)+len(deviceIdsFilter))
	for _, r := range resourceIDsFilter {
		res := commands.ResourceIdFromString(r)

		if rd.userDeviceIds.HasOneOf(res.GetDeviceId()) {
			internalResourceIDsFilter = append(internalResourceIDsFilter, res)
		}
	}
	for _, deviceID := range deviceIdsFilter {
		if rd.userDeviceIds.HasOneOf(deviceID) {
			internalResourceIDsFilter = append(internalResourceIDsFilter, commands.NewResourceID(deviceID, ""))
		}
	}
	if len(internalResourceIDsFilter) == 0 {
		if len(resourceIDsFilter) > 0 || len(deviceIdsFilter) > 0 {
			return nil, status.Errorf(codes.NotFound, "resource ids filter doesn't match any resources")
		}
		internalResourceIDsFilter = make([]*commands.ResourceId, 0, len(rd.userDeviceIds))
		for userDeviceID := range rd.userDeviceIds {
			internalResourceIDsFilter = append(internalResourceIDsFilter, commands.NewResourceID(userDeviceID, ""))
		}
	}

	resources, err := rd.projection.GetResourcesWithLinks(ctx, internalResourceIDsFilter, mapTypeFilter)
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, status.Errorf(codes.NotFound, "not found")
	}
	return resources, err
}

func (rd *ResourceShadow) GetResources(req *pb.GetResourcesRequest, srv pb.GrpcGateway_GetResourcesServer) error {
	contentEncoder, err := commands.GetContentEncoder(grpc.AcceptContentFromMD(srv.Context()))
	if err != nil {
		return err
	}

	resources, err := rd.filterResources(srv.Context(), req.GetResourceIdsFilter(), req.GetDeviceIdsFilter(), req.GetTypeFilter())
	if err != nil {
		return err
	}

	for _, deviceResources := range resources {
		for _, resource := range deviceResources {
			val, err := toResourceValue(resource, contentEncoder)
			if err != nil {
				log.Errorf("cannot send resource(%+v): %v", val.Data, err)
				continue
			}
			err = srv.Send(val)
			if err != nil {
				return status.Errorf(codes.Canceled, "cannot send resource value %v: %v", val, err)
			}
		}
	}
	return nil
}

func (rd *ResourceShadow) GetPendingCommands(req *pb.GetPendingCommandsRequest, srv pb.GrpcGateway_GetPendingCommandsServer) error {
	contentEncoder, err := commands.GetContentEncoder(grpc.AcceptContentFromMD(srv.Context()))
	if err != nil {
		return err
	}

	filterCmds := filterPendingsCommandsToBitmask(req.GetCommandsFilter())
	if filterCmds&filterBitmaskDeviceMetadataUpdatePending > 0 && len(req.GetResourceIdsFilter()) == 0 && len(req.GetTypeFilter()) == 0 {
		deviceIDs := filterDevices(rd.userDeviceIds, req.GetDeviceIdsFilter())
		devicesMetadata, err := rd.projection.GetDevicesMetadata(srv.Context(), deviceIDs)
		if err != nil {
			return err
		}
		for _, deviceMetadata := range devicesMetadata {
			for _, pendingCmd := range deviceMetadata.GetUpdatePendings() {
				err = srv.Send(&pb.PendingCommand{
					Command: &pb.PendingCommand_DeviceMetadataUpdatePending{
						DeviceMetadataUpdatePending: pendingCmd,
					},
				})
				if err != nil {
					return status.Errorf(codes.Canceled, "cannot send device metadata update pending command %v: %v", pendingCmd, err)
				}
			}
		}
	}

	resources, err := rd.filterResources(srv.Context(), req.GetResourceIdsFilter(), req.GetDeviceIdsFilter(), req.GetTypeFilter())
	if err != nil {
		return err
	}

	for _, deviceResources := range resources {
		for _, resource := range deviceResources {
			for _, pendingCmd := range toPendingCommands(resource, filterCmds, contentEncoder) {
				err = srv.Send(pendingCmd)
				if err != nil {
					return status.Errorf(codes.Canceled, "cannot send pending command %v: %v", pendingCmd, err)
				}
			}
		}
	}
	return nil
}

func filterMetadataByUserFilters(resources map[string]map[string]*Resource, devicesMetadata map[string]*events.DeviceMetadataSnapshotTaken, req *pb.GetDevicesMetadataRequest) (map[string]*events.DeviceMetadataSnapshotTaken, error) {
	result := make(map[string]*events.DeviceMetadataSnapshotTaken)
	typeFilter := make(kitStrings.Set)
	typeFilter.Add(req.TypeFilter...)
	for deviceID, resources := range resources {
		for _, resource := range resources {
			if !hasMatchingType(resource.Resource.GetResourceTypes(), typeFilter) {
				continue
			}
			d, ok := devicesMetadata[deviceID]
			if ok {
				result[deviceID] = d
			}
		}
	}

	return result, nil
}

func (rd *ResourceShadow) GetDevicesMetadata(req *pb.GetDevicesMetadataRequest, srv pb.GrpcGateway_GetDevicesMetadataServer) error {
	deviceIDs := filterDevices(rd.userDeviceIds, req.DeviceIdsFilter)
	resourceIdsFilter := make([]*commands.ResourceId, 0, 64)
	for deviceID := range deviceIDs {
		resourceIdsFilter = append(resourceIdsFilter, commands.NewResourceID(deviceID, "/oic/d"), commands.NewResourceID(deviceID, commands.StatusHref))
	}

	resources, err := rd.projection.GetResourcesWithLinks(srv.Context(), resourceIdsFilter, nil)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get resources by device ids: %v", err)
	}

	devicesMetadata, err := rd.projection.GetDevicesMetadata(srv.Context(), deviceIDs)
	if err != nil {
		return err
	}

	devicesMetadata, err = filterMetadataByUserFilters(resources, devicesMetadata, req)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot filter devices metadata by type: %v", err)
	}

	if len(devicesMetadata) == 0 {
		return status.Errorf(codes.NotFound, "not found")
	}

	for _, deviceMetadata := range devicesMetadata {
		err = srv.Send(deviceMetadata.GetDeviceMetadataUpdated())
		if err != nil {
			return status.Errorf(codes.Canceled, "cannot send devices metadata %v: %v", deviceMetadata, err)
		}
	}
	return nil
}
