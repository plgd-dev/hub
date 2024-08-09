package grpc

import (
	"context"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *DeviceProvisionServiceServer) GetHubs(req *pb.GetHubsRequest, srv pb.DeviceProvisionService_GetHubsServer) error {
	owner, err := grpc.OwnerFromTokenMD(srv.Context(), d.ownerClaim)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "cannot get owner: %v", err)
	}
	return d.store.LoadHubs(srv.Context(), owner, req, func(ctx context.Context, iter store.HubIter) (err error) {
		for {
			var g pb.Hub
			if ok := iter.Next(ctx, &g); !ok {
				return iter.Err()
			}
			if err = srv.Send(&g); err != nil {
				return err
			}
		}
	})
}
