package grpc

import (
	"context"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *DeviceProvisionServiceServer) DeleteProvisioningRecords(ctx context.Context, req *pb.DeleteProvisioningRecordsRequest) (*pb.DeleteProvisioningRecordsResponse, error) {
	owner, err := grpc.OwnerFromTokenMD(ctx, d.ownerClaim)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "cannot get owner: %v", err)
	}
	count, err := d.store.DeleteProvisioningRecords(ctx, owner, &pb.GetProvisioningRecordsRequest{
		IdFilter:                req.GetIdFilter(),
		DeviceIdFilter:          req.GetDeviceIdFilter(),
		EnrollmentGroupIdFilter: req.GetEnrollmentGroupIdFilter(),
	})
	if err != nil {
		return nil, err
	}
	return &pb.DeleteProvisioningRecordsResponse{
		Count: count,
	}, nil
}
