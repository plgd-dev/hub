package grpc

import (
	"context"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *DeviceProvisionServiceServer) DeleteHubs(ctx context.Context, req *pb.DeleteHubsRequest) (*pb.DeleteHubsResponse, error) {
	owner, err := grpc.OwnerFromTokenMD(ctx, d.ownerClaim)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "cannot get owner: %v", err)
	}
	count, err := d.store.DeleteHubs(ctx, owner, &pb.GetHubsRequest{IdFilter: req.GetIdFilter()})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "hub('%v'): %v", req.GetIdFilter(), err)
	}
	return &pb.DeleteHubsResponse{
		Count: count,
	}, nil
}
