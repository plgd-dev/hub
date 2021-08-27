package subscription

import (
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
)

type FilterBitmask uint64

const (
	FilterBitmaskResourceCreatePending       FilterBitmask = 1
	FilterBitmaskResourceCreated             FilterBitmask = 1 << 1
	FilterBitmaskResourceRetrievePending     FilterBitmask = 1 << 2
	FilterBitmaskResourceRetrieved           FilterBitmask = 1 << 3
	FilterBitmaskResourceUpdatePending       FilterBitmask = 1 << 4
	FilterBitmaskResourceUpdated             FilterBitmask = 1 << 5
	FilterBitmaskResourceDeletePending       FilterBitmask = 1 << 6
	FilterBitmaskResourceDeleted             FilterBitmask = 1 << 7
	FilterBitmaskDeviceMetadataUpdatePending FilterBitmask = 1 << 8
	FilterBitmaskDeviceMetadataUpdated       FilterBitmask = 1 << 9
	FilterBitmaskDeviceRegistered            FilterBitmask = 1 << 10
	FilterBitmaskDeviceUnregistered          FilterBitmask = 1 << 11
	FilterBitmaskResourceChanged             FilterBitmask = 1 << 12
	FilterBitmaskResourcesPublished          FilterBitmask = 1 << 13
	FilterBitmaskResourcesUnpublished        FilterBitmask = 1 << 14
)

var pendingCommandToBitmask = map[pb.GetPendingCommandsRequest_Command]FilterBitmask{
	pb.GetPendingCommandsRequest_RESOURCE_CREATE:        FilterBitmaskResourceCreatePending,
	pb.GetPendingCommandsRequest_RESOURCE_RETRIEVE:      FilterBitmaskResourceRetrievePending,
	pb.GetPendingCommandsRequest_RESOURCE_UPDATE:        FilterBitmaskResourceUpdatePending,
	pb.GetPendingCommandsRequest_RESOURCE_DELETE:        FilterBitmaskResourceDeletePending,
	pb.GetPendingCommandsRequest_DEVICE_METADATA_UPDATE: FilterBitmaskDeviceMetadataUpdatePending,
}

func FilterPendingCommandToBitmask(f pb.GetPendingCommandsRequest_Command) FilterBitmask {
	return pendingCommandToBitmask[f]
}

func FilterPendingsCommandsToBitmask(commandFilter []pb.GetPendingCommandsRequest_Command) FilterBitmask {
	bitmask := FilterBitmask(0)
	if len(commandFilter) == 0 {
		for _, bit := range pendingCommandToBitmask {
			bitmask |= bit
		}
	} else {
		for _, f := range commandFilter {
			bitmask |= FilterPendingCommandToBitmask(f)
		}
	}
	return bitmask
}

var bitmaskTopendingCommands = map[FilterBitmask]pb.GetPendingCommandsRequest_Command{
	FilterBitmaskResourceCreatePending:       pb.GetPendingCommandsRequest_RESOURCE_CREATE,
	FilterBitmaskResourceRetrievePending:     pb.GetPendingCommandsRequest_RESOURCE_RETRIEVE,
	FilterBitmaskResourceUpdatePending:       pb.GetPendingCommandsRequest_RESOURCE_UPDATE,
	FilterBitmaskResourceDeletePending:       pb.GetPendingCommandsRequest_RESOURCE_DELETE,
	FilterBitmaskDeviceMetadataUpdatePending: pb.GetPendingCommandsRequest_DEVICE_METADATA_UPDATE,
}

func BitmaskToFilterPendingsCommands(bitmask FilterBitmask) []pb.GetPendingCommandsRequest_Command {
	if bitmask == 0 {
		return nil
	}
	res := make([]pb.GetPendingCommandsRequest_Command, 0, 5)
	for bit, val := range bitmaskTopendingCommands {
		if bitmask&bit == 0 {
			continue
		}
		res = append(res, val)
	}
	return res
}

func EventFilterToBitmask(f pb.SubscribeToEvents_CreateSubscription_Event) FilterBitmask {
	bitmask := FilterBitmask(0)
	switch f {
	case pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATE_PENDING:
		bitmask |= FilterBitmaskResourceCreatePending
	case pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATED:
		bitmask |= FilterBitmaskResourceCreated
	case pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING:
		bitmask |= FilterBitmaskResourceRetrievePending
	case pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVED:
		bitmask |= FilterBitmaskResourceRetrieved
	case pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING:
		bitmask |= FilterBitmaskResourceUpdatePending
	case pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATED:
		bitmask |= FilterBitmaskResourceUpdated
	case pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETE_PENDING:
		bitmask |= FilterBitmaskResourceDeletePending
	case pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETED:
		bitmask |= FilterBitmaskResourceDeleted
	case pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATE_PENDING:
		bitmask |= FilterBitmaskDeviceMetadataUpdatePending
	case pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED:
		bitmask |= FilterBitmaskDeviceMetadataUpdated
	case pb.SubscribeToEvents_CreateSubscription_REGISTERED:
		bitmask |= FilterBitmaskDeviceRegistered
	case pb.SubscribeToEvents_CreateSubscription_UNREGISTERED:
		bitmask |= FilterBitmaskDeviceUnregistered
	case pb.SubscribeToEvents_CreateSubscription_RESOURCE_PUBLISHED:
		bitmask |= FilterBitmaskResourcesPublished
	case pb.SubscribeToEvents_CreateSubscription_RESOURCE_UNPUBLISHED:
		bitmask |= FilterBitmaskResourcesUnpublished
	case pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED:
		bitmask |= FilterBitmaskResourceChanged
	}
	return bitmask
}

func EventsFilterToBitmask(commandFilter []pb.SubscribeToEvents_CreateSubscription_Event) FilterBitmask {
	bitmask := FilterBitmask(0)
	if len(commandFilter) == 0 {
		bitmask = FilterBitmask(0xffffffff)
	} else {
		for _, f := range commandFilter {
			bitmask |= EventFilterToBitmask(f)
		}
	}
	return bitmask
}
