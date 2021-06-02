package service

import (
	"context"

	"github.com/plgd-dev/kit/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
)

func toResourceValue(resource *Resource) *pb.ResourceValue {
	return &pb.ResourceValue{
		ResourceId: &commands.ResourceId{
			Href:     resource.Resource.GetHref(),
			DeviceId: resource.Resource.GetDeviceId(),
		},
		Content: pb.RAContent2Content(resource.GetContent()),
		Types:   resource.Resource.GetResourceTypes(),
		Status:  pb.RAStatus2Status(resource.GetStatus()),
	}
}

type commandsFilterBitmask int

const (
	commandsFilterBitmaskCreate               commandsFilterBitmask = 1
	commandsFilterBitmaskRetrieve             commandsFilterBitmask = 2
	commandsFilterBitmaskUpdate               commandsFilterBitmask = 4
	commandsFilterBitmaskDelete               commandsFilterBitmask = 8
	commandsFilterBitmaskDeviceMetadataUpdate commandsFilterBitmask = 16
)

func filterPendingCommandToBitmask(f pb.RetrievePendingCommandsRequest_Command) commandsFilterBitmask {
	bitmask := commandsFilterBitmask(0)
	switch f {
	case pb.RetrievePendingCommandsRequest_RESOURCE_CREATE:
		bitmask |= commandsFilterBitmaskCreate
	case pb.RetrievePendingCommandsRequest_RESOURCE_RETRIEVE:
		bitmask |= commandsFilterBitmaskRetrieve
	case pb.RetrievePendingCommandsRequest_RESOURCE_UPDATE:
		bitmask |= commandsFilterBitmaskUpdate
	case pb.RetrievePendingCommandsRequest_RESOURCE_DELETE:
		bitmask |= commandsFilterBitmaskDelete
	case pb.RetrievePendingCommandsRequest_DEVICE_METADATA_UPDATE:
		bitmask |= commandsFilterBitmaskDeviceMetadataUpdate
	}
	return bitmask
}

func filterPendingsCommandsToBitmask(commandsFilter []pb.RetrievePendingCommandsRequest_Command) commandsFilterBitmask {
	bitmask := commandsFilterBitmask(0)
	if len(commandsFilter) == 0 {
		bitmask = commandsFilterBitmaskCreate | commandsFilterBitmaskRetrieve | commandsFilterBitmaskUpdate | commandsFilterBitmaskDelete | commandsFilterBitmaskDeviceMetadataUpdate
	} else {
		for _, f := range commandsFilter {
			bitmask |= filterPendingCommandToBitmask(f)
		}
	}
	return bitmask
}

func toPendingCommands(resource *Resource, commandsFilter commandsFilterBitmask) []*pb.PendingCommand {
	if resource.projection == nil {
		return nil
	}
	pendingCmds := make([]*pb.PendingCommand, 0, 32)
	if commandsFilter&commandsFilterBitmaskCreate > 0 {
		for _, pendingCmd := range resource.projection.resourceCreatePendings {
			pendingCmds = append(pendingCmds, &pb.PendingCommand{
				Command: &pb.PendingCommand_ResourceCreatePending{
					ResourceCreatePending: pendingCmd,
				},
			})
		}
	}
	if commandsFilter&commandsFilterBitmaskRetrieve > 0 {
		for _, pendingCmd := range resource.projection.resourceRetrievePendings {
			pendingCmds = append(pendingCmds, &pb.PendingCommand{
				Command: &pb.PendingCommand_ResourceRetrievePending{
					ResourceRetrievePending: pendingCmd,
				},
			})
		}
	}
	if commandsFilter&commandsFilterBitmaskUpdate > 0 {
		for _, pendingCmd := range resource.projection.resourceUpdatePendings {
			pendingCmds = append(pendingCmds, &pb.PendingCommand{
				Command: &pb.PendingCommand_ResourceUpdatePending{
					ResourceUpdatePending: pendingCmd,
				},
			})
		}
	}
	if commandsFilter&commandsFilterBitmaskDelete > 0 {
		for _, pendingCmd := range resource.projection.resourceDeletePendings {
			pendingCmds = append(pendingCmds, &pb.PendingCommand{
				Command: &pb.PendingCommand_ResourceDeletePending{
					ResourceDeletePending: pendingCmd,
				},
			})
		}
	}
	return pendingCmds
}

type ResourceShadow struct {
	projection    *Projection
	userDeviceIds strings.Set
}

func NewResourceShadow(projection *Projection, deviceIds []string) *ResourceShadow {
	mapDeviceIds := make(strings.Set)
	mapDeviceIds.Add(deviceIds...)

	return &ResourceShadow{projection: projection, userDeviceIds: mapDeviceIds}
}

func (rd *ResourceShadow) filterResources(ctx context.Context, resourceIDsFilter []*commands.ResourceId, deviceIdsFilter, typeFilter []string) (map[string]map[string]*Resource, error) {
	mapTypeFilter := make(strings.Set)
	mapTypeFilter.Add(typeFilter...)

	internalResourceIDsFilter := make([]*commands.ResourceId, 0, len(resourceIDsFilter)+len(deviceIdsFilter))
	for _, res := range resourceIDsFilter {
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

func (rd *ResourceShadow) RetrieveResourcesValues(req *pb.RetrieveResourcesValuesRequest, srv pb.GrpcGateway_RetrieveResourcesValuesServer) error {
	resources, err := rd.filterResources(srv.Context(), req.GetResourceIdsFilter(), req.GetDeviceIdsFilter(), req.GetTypeFilter())
	if err != nil {
		return err
	}

	for _, deviceResources := range resources {
		for _, resource := range deviceResources {
			val := toResourceValue(resource)
			err = srv.Send(val)
			if err != nil {
				return status.Errorf(codes.Canceled, "cannot send resource value %v: %v", val, err)
			}
		}
	}
	return nil
}

func (rd *ResourceShadow) RetrievePendingCommands(req *pb.RetrievePendingCommandsRequest, srv pb.GrpcGateway_RetrievePendingCommandsServer) error {
	filterCmds := filterPendingsCommandsToBitmask(req.GetCommandsFilter())
	if filterCmds&commandsFilterBitmaskDeviceMetadataUpdate > 0 && len(req.GetResourceIdsFilter()) == 0 && len(req.GetTypeFilter()) == 0 {
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
			for _, pendingCmd := range toPendingCommands(resource, filterCmds) {
				err = srv.Send(pendingCmd)
				if err != nil {
					return status.Errorf(codes.Canceled, "cannot send pending command %v: %v", pendingCmd, err)
				}
			}
		}
	}
	return nil
}

func (rd *ResourceShadow) RetrieveDevicesMetadata(req *pb.RetrieveDevicesMetadataRequest, srv pb.GrpcGateway_RetrieveDevicesMetadataServer) error {
	deviceIDs := filterDevices(rd.userDeviceIds, req.DeviceIdsFilter)
	devicesMetadata, err := rd.projection.GetDevicesMetadata(srv.Context(), deviceIDs)
	if err != nil {
		return err
	}
	for _, deviceMetadata := range devicesMetadata {
		err = srv.Send(deviceMetadata)
		if err != nil {
			return status.Errorf(codes.Canceled, "cannot send devices metadata %v: %v", deviceMetadata, err)
		}
	}
	return nil
}
