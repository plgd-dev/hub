package service

import (
	"context"

	pbCA "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
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
	deviceIDs, _ := strings.Split(strings.Unique(req.DeviceIdFilter), func(s string) bool {
		return s != ""
	})

	deleteAllOwned := len(deviceIDs) == 0
	// ResourceAggregate
	cmdRA := commands.DeleteDevicesRequest{DeviceIds: deviceIDs}
	respRA, err := r.resourceAggregateClient.DeleteDevices(ctx, &cmdRA)
	if err != nil {
		return nil, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot delete devices from ResourceAggregate: %v", err)
	}
	if !deleteAllOwned {
		_, notDeleted := partitionDeletedDevices(deviceIDs, respRA.GetDeviceIds())
		if len(notDeleted) > 0 {
			for _, deviceID := range notDeleted {
				r.logger.Debugf("failed to delete device('%v') in ResourceAggregate", deviceID)
			}
		}
	}

	// CertificateAuthority
	respCA, err := r.certificateAuthorityClient.DeleteSigningRecords(ctx, &pbCA.DeleteSigningRecordsRequest{
		DeviceIdFilter: deviceIDs,
	})
	if err != nil {
		return nil, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot delete certificates linked to devices %v from CertificateAuthority: %v", deviceIDs, err)
	}
	r.logger.Debugf("certificate records(num: %v) linked to %v devices has been deleted", respCA.GetCount(), deviceIDs)

	// IdentityStore
	cmdAS := pbIS.DeleteDevicesRequest{
		DeviceIds: deviceIDs,
	}
	respIS, err := r.idClient.DeleteDevices(ctx, &cmdAS)
	if err != nil {
		return nil, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot delete devices from identity-store: %v", err)
	}
	if !deleteAllOwned {
		_, notDeleted := partitionDeletedDevices(deviceIDs, respIS.GetDeviceIds())
		if len(notDeleted) > 0 {
			for _, deviceID := range notDeleted {
				r.logger.Debugf("failed to delete device('%v') from identity-store", deviceID)
			}
		}
	}

	return &pb.DeleteDevicesResponse{
		DeviceIds: respIS.GetDeviceIds(),
	}, nil
}
