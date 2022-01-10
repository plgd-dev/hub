package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/v2/strings"
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

// Delete documents from events database for devices selected by query
//
// Using empty deviceIdFilter in DeleteDevicesRequest is interpreting as requesting
// to delete all documents for devices owned by the user.
//
// Function returns error or a non-empty DeleteDevicesResponse message, where the DeviceIds
// field is filled with list of device ids. The list is an intersection of the list provided
// by DeleteDevicesRequest and device ids owned by the user (ie. from the original list of device
// ids it filters out devices that are not owned by the user).
func (r RequestHandler) DeleteDevices(ctx context.Context, request *commands.DeleteDevicesRequest) (*commands.DeleteDevicesResponse, error) {
	deviceIds := getUniqueDeviceIdsFromDeleteRequest(request)
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

	if err := r.eventstore.Delete(ctx, queries); err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete devices: %v", err))
	}

	return &commands.DeleteDevicesResponse{
		DeviceIds:    ownedDevices,
		AuditContext: commands.NewAuditContext(owner, ""),
	}, nil
}
