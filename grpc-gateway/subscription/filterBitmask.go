package subscription

import (
	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
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

func IsFilteredBit(filteredEventTypes FilterBitmask, bit FilterBitmask) bool {
	return filteredEventTypes&bit != 0
}

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

var eventsToBitmask = map[pb.SubscribeToEvents_CreateSubscription_Event]FilterBitmask{
	pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATE_PENDING:        FilterBitmaskResourceCreatePending,
	pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATED:               FilterBitmaskResourceCreated,
	pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING:      FilterBitmaskResourceRetrievePending,
	pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVED:             FilterBitmaskResourceRetrieved,
	pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING:        FilterBitmaskResourceUpdatePending,
	pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATED:               FilterBitmaskResourceUpdated,
	pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETE_PENDING:        FilterBitmaskResourceDeletePending,
	pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETED:               FilterBitmaskResourceDeleted,
	pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATE_PENDING: FilterBitmaskDeviceMetadataUpdatePending,
	pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED:        FilterBitmaskDeviceMetadataUpdated,
	pb.SubscribeToEvents_CreateSubscription_REGISTERED:                     FilterBitmaskDeviceRegistered,
	pb.SubscribeToEvents_CreateSubscription_UNREGISTERED:                   FilterBitmaskDeviceUnregistered,
	pb.SubscribeToEvents_CreateSubscription_RESOURCE_PUBLISHED:             FilterBitmaskResourcesPublished,
	pb.SubscribeToEvents_CreateSubscription_RESOURCE_UNPUBLISHED:           FilterBitmaskResourcesUnpublished,
	pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED:               FilterBitmaskResourceChanged,
}

func EventFilterToBitmask(f pb.SubscribeToEvents_CreateSubscription_Event) FilterBitmask {
	return eventsToBitmask[f]
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
