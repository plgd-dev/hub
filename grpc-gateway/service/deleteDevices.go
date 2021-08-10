package service

import (
	"context"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/kit/strings"
	"google.golang.org/grpc/codes"
)

// Split array into two based on whether the array item is contained in the expected array or not
func partitionDeletedDevices(expected, actual []string) ([]string, []string) {
	cache := make(map[string]struct{})
	for _, v := range actual {
		cache[v] = struct{}{}
	}

	deleted := make(strings.Set)
	notDeleted := make(strings.Set)
	for _, v := range expected {
		_, ok := cache[v]
		if ok {
			deleted.Add(v)
		} else {
			notDeleted.Add(v)
		}
	}

	return deleted.ToSlice(), notDeleted.ToSlice()
}

func (r *RequestHandler) DeleteDevices(ctx context.Context, req *pb.DeleteDevicesRequest) (*pb.DeleteDevicesResponse, error) {
	cmdRA, err := req.ToRACommand()
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete devices: %v", err))
	}
	deleteAllOwned := len(cmdRA.DeviceIds) == 0
	respRA, err := r.resourceAggregateClient.DeleteDevices(ctx, cmdRA)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete devices from ResourceAggregate: %v", err))
	}

	var deleted, notDeleted []string
	if deleteAllOwned {
		deleted = respRA.DeviceIds
	} else {
		deleted, notDeleted = partitionDeletedDevices(cmdRA.GetDeviceIds(), respRA.GetDeviceIds())
		if len(notDeleted) > 0 {
			for _, deviceId := range notDeleted {
				log.Errorf("failed to delete device('%v') in ResourceAggregate", deviceId)
			}
		}
	}

	if len(deleted) == 0 {
		return &pb.DeleteDevicesResponse{
			DeviceIds: deleted,
		}, nil
	}

	cmdAS := pbAS.DeleteDevicesRequest{
		DeviceIds: deleted,
	}
	respAS, err := r.authorizationClient.DeleteDevices(ctx, &cmdAS)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete devices in Authorization service: %v", err))
	}
	deleted, notDeleted = partitionDeletedDevices(cmdAS.GetDeviceIds(), respAS.GetDeviceIds())
	if len(notDeleted) > 0 {
		for _, deviceId := range notDeleted {
			log.Errorf("failed to delete device('%v') in Authorization service", deviceId)
		}
	}

	return &pb.DeleteDevicesResponse{
		DeviceIds: deleted,
	}, nil
}
