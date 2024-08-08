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

const (
	errEnrollmentGroupFmt         = "enrollment group('%v'): %v"
	errEnrollmentGroupNotFoundFmt = "enrollment group('%v'): not found"
)

func (d *DeviceProvisionServiceServer) loadEnrollmentGroup(ctx context.Context, owner, id string) (*pb.EnrollmentGroup, error) {
	var res pb.EnrollmentGroup
	var ok bool
	err := d.store.LoadEnrollmentGroups(ctx, owner, &pb.GetEnrollmentGroupsRequest{IdFilter: []string{id}}, func(ctx context.Context, iter store.EnrollmentGroupIter) (err error) {
		ok = iter.Next(ctx, &res)
		return iter.Err()
	})
	if err != nil {
		if errors.Is(err, mongo.ErrNilDocument) {
			return nil, status.Errorf(codes.NotFound, errEnrollmentGroupNotFoundFmt, id)
		}
		return nil, status.Errorf(codes.InvalidArgument, errEnrollmentGroupFmt, id, err)
	}
	if !ok {
		return nil, status.Errorf(codes.NotFound, errEnrollmentGroupNotFoundFmt, id)
	}
	return &res, nil
}

func (d *DeviceProvisionServiceServer) UpdateEnrollmentGroup(ctx context.Context, req *pb.UpdateEnrollmentGroupRequest) (*pb.EnrollmentGroup, error) {
	x509Attestation := req.GetEnrollmentGroup().GetAttestationMechanism().GetX509()
	if x509Attestation != nil {
		err := x509Attestation.Validate()
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, errEnrollmentGroupFmt, req.GetId(), err)
		}
	}
	owner, err := grpc.OwnerFromTokenMD(ctx, d.ownerClaim)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "cannot get owner: %v", err)
	}

	err = d.store.UpdateEnrollmentGroup(ctx, owner, &pb.EnrollmentGroup{
		Id:                   req.GetId(),
		Owner:                owner,
		AttestationMechanism: req.GetEnrollmentGroup().GetAttestationMechanism(),
		HubIds:               req.GetEnrollmentGroup().GetHubIds(),
		PreSharedKey:         req.GetEnrollmentGroup().GetPreSharedKey(),
		Name:                 req.GetEnrollmentGroup().GetName(),
	})
	if err != nil {
		if errors.Is(err, mongo.ErrNilDocument) {
			return nil, status.Errorf(codes.NotFound, errEnrollmentGroupNotFoundFmt, req.GetId())
		}
		return nil, status.Errorf(codes.InvalidArgument, errEnrollmentGroupFmt, req.GetId(), err)
	}
	return d.loadEnrollmentGroup(ctx, owner, req.GetId())
}
