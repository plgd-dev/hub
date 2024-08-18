package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *DeviceProvisionServiceServer) CreateEnrollmentGroup(ctx context.Context, req *pb.CreateEnrollmentGroupRequest) (*pb.EnrollmentGroup, error) {
	owner, err := grpc.OwnerFromTokenMD(ctx, d.ownerClaim)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "cannot get owner: %v", err)
	}
	g := &pb.EnrollmentGroup{
		Id:                   uuid.NewString(),
		Owner:                owner,
		AttestationMechanism: req.GetAttestationMechanism(),
		HubIds:               req.GetHubIds(),
		PreSharedKey:         req.GetPreSharedKey(),
		Name:                 req.GetName(),
	}
	err = d.store.CreateEnrollmentGroup(ctx, owner, g)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "enrollment group('%v'): %v", g.GetId(), err)
	}
	return g, nil
}
