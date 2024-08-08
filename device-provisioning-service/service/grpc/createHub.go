package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *DeviceProvisionServiceServer) CreateHub(ctx context.Context, req *pb.CreateHubRequest) (*pb.Hub, error) {
	owner, err := grpc.OwnerFromTokenMD(ctx, d.ownerClaim)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "cannot get owner: %v", err)
	}

	h := &pb.Hub{
		Id:                   uuid.NewString(),
		HubId:                req.GetHubId(),
		Gateways:             req.GetGateways(),
		CertificateAuthority: req.GetCertificateAuthority(),
		Authorization:        req.GetAuthorization(),
		Name:                 req.GetName(),
		Owner:                owner,
	}
	err = d.store.CreateHub(ctx, owner, h)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "hub('%v'): %v", req.GetHubId(), err)
	}
	return h, nil
}
