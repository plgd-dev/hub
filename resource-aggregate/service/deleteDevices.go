package service

import (
	"context"

	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/strings"
	"google.golang.org/grpc/codes"
)

func getUniqueDeviceIdsFromDeleteRequest(request *commands.DeleteDevicesRequest) []string {
	deviceIds := make(strings.Set)
	for _, deviceId := range request.DeviceIds {
		if deviceId != "" {
			deviceIds.Add(deviceId)
		}
	}
	return deviceIds.ToSlice()
}

func (r RequestHandler) DeleteDevices(ctx context.Context, request *commands.DeleteDevicesRequest) (*commands.DeleteDevicesResponse, error) {
	deviceIds := getUniqueDeviceIdsFromDeleteRequest(request)
	if len(deviceIds) == 0 {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot delete devices: invalid request"))
	}

	owner, ownedDevices, err := r.getOwnedDevices(ctx, deviceIds)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}

	if len(ownedDevices) == 0 {
		return &commands.DeleteDevicesResponse{
			DeviceIds:    nil,
			AuditContext: commands.NewAuditContext(owner, ""),
		}, nil
	}

	queries := make([]eventstore.DeleteQuery, len(ownedDevices))
	for i, dev := range ownedDevices {
		queries[i].GroupID = dev
	}

	deletedDeviceIds, err := r.eventstore.Delete(ctx, queries)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete devices: %v", err))
	}

	return &commands.DeleteDevicesResponse{
		DeviceIds:    deletedDeviceIds,
		AuditContext: commands.NewAuditContext(owner, ""),
	}, nil
}
