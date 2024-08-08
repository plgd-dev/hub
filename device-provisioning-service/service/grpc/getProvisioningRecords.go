package grpc

import (
	"context"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *DeviceProvisionServiceServer) GetProvisioningRecords(req *pb.GetProvisioningRecordsRequest, srv pb.DeviceProvisionService_GetProvisioningRecordsServer) error {
	owner, err := grpc.OwnerFromTokenMD(srv.Context(), d.ownerClaim)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "cannot get owner: %v", err)
	}
	return d.store.LoadProvisioningRecords(srv.Context(), owner, req, func(ctx context.Context, iter store.ProvisioningRecordIter) (err error) {
		for {
			var sub pb.ProvisioningRecord
			if ok := iter.Next(ctx, &sub); !ok {
				return iter.Err()
			}
			if err = srv.Send(&sub); err != nil {
				return err
			}
		}
	})
}
