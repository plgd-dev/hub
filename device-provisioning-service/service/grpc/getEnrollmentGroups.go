package grpc

import (
	"context"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *DeviceProvisionServiceServer) GetEnrollmentGroups(req *pb.GetEnrollmentGroupsRequest, srv pb.DeviceProvisionService_GetEnrollmentGroupsServer) error {
	owner, err := grpc.OwnerFromTokenMD(srv.Context(), d.ownerClaim)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "cannot get owner: %v", err)
	}
	return d.store.LoadEnrollmentGroups(srv.Context(), owner, req, func(ctx context.Context, iter store.EnrollmentGroupIter) (err error) {
		for {
			var g pb.EnrollmentGroup
			if ok := iter.Next(ctx, &g); !ok {
				return iter.Err()
			}
			if err = srv.Send(&g); err != nil {
				return err
			}
		}
	})
}
