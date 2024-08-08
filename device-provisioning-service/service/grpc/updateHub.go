package grpc

import (
	"context"
	"errors"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const errHubNotFoundFmt = "hub('%v'): not found"

func (d *DeviceProvisionServiceServer) loadHub(ctx context.Context, owner string, id string) (*pb.Hub, error) {
	var res pb.Hub
	var ok bool
	err := d.store.LoadHubs(ctx, owner, &pb.GetHubsRequest{IdFilter: []string{id}}, func(ctx context.Context, iter store.HubIter) (err error) {
		ok = iter.Next(ctx, &res)
		return iter.Err()
	})
	if err != nil {
		if errors.Is(err, mongo.ErrNilDocument) {
			return nil, status.Errorf(codes.NotFound, errHubNotFoundFmt, id)
		}
		return nil, status.Errorf(codes.InvalidArgument, "hub('%v'): %v", id, err)
	}
	if !ok {
		return nil, status.Errorf(codes.NotFound, errHubNotFoundFmt, id)
	}
	return &res, nil
}

func (d *DeviceProvisionServiceServer) UpdateHub(ctx context.Context, req *pb.UpdateHubRequest) (*pb.Hub, error) {
	owner, err := grpc.OwnerFromTokenMD(ctx, d.ownerClaim)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "cannot get owner: %v", err)
	}

	err = d.store.UpdateHub(ctx, owner, &pb.Hub{
		Id:                   req.GetId(),
		HubId:                req.GetHub().GetHubId(),
		Gateways:             req.GetHub().GetGateways(),
		CertificateAuthority: req.GetHub().GetCertificateAuthority(),
		Authorization:        req.GetHub().GetAuthorization(),
		Name:                 req.GetHub().GetName(),
		Owner:                owner,
	})
	if err != nil {
		if errors.Is(err, mongo.ErrNilDocument) {
			return nil, status.Errorf(codes.NotFound, errHubNotFoundFmt, req.GetId())
		}
		return nil, status.Errorf(codes.InvalidArgument, "hub('%v'): %v", req.GetId(), err)
	}
	return d.loadHub(ctx, owner, req.GetId())
}
