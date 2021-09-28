package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	pbIS "github.com/plgd-dev/cloud/identity/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/strings"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

// Split array into two based on whether the array item is contained in the expected array or not
func partitionDeletedDevices(expected, actual []string) ([]string, []string) {
	cache := make(map[string]struct{})
	for _, v := range actual {
		cache[v] = struct{}{}
	}
	contains := func(s string) bool {
		_, ok := cache[s]
		return ok
	}

	return strings.Split(expected, contains)
}

func (r *RequestHandler) DeleteDevices(ctx context.Context, req *pb.DeleteDevicesRequest) (*pb.DeleteDevicesResponse, error) {
	// get unique non-empty ids
	deviceIds, _ := strings.Split(strings.Unique(req.DeviceIdFilter), func(s string) bool {
		return s != ""
	})

	deleteAllOwned := len(deviceIds) == 0
	// ResourceAggregate
	cmdRA := commands.DeleteDevicesRequest{DeviceIds: deviceIds}
	respRA, err := r.resourceAggregateClient.DeleteDevices(ctx, &cmdRA)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete devices from ResourceAggregate: %v", err))
	}
	if !deleteAllOwned {
		_, notDeleted := partitionDeletedDevices(deviceIds, respRA.GetDeviceIds())
		if len(notDeleted) > 0 {
			for _, deviceId := range notDeleted {
				log.Debugf("failed to delete device('%v') in ResourceAggregate", deviceId)
			}
		}
	}

	// Identity service
	cmdAS := pbIS.DeleteDevicesRequest{
		DeviceIds: deviceIds,
	}
	respIS, err := r.idClient.DeleteDevices(ctx, &cmdAS)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete devices identity service: %v", err))
	}
	if !deleteAllOwned {
		_, notDeleted := partitionDeletedDevices(deviceIds, respIS.GetDeviceIds())
		if len(notDeleted) > 0 {
			for _, deviceId := range notDeleted {
				log.Debugf("failed to delete device('%v') in identity service", deviceId)
			}
		}
	}

	return &pb.DeleteDevicesResponse{
		DeviceIds: respIS.GetDeviceIds(),
	}, nil
}
