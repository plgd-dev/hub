package service

import (
	"context"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"

	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RemoveDevice remove a device from user. It is used by cloud2cloud connector.
func (s *Service) RemoveDevice(ctx context.Context, request *pb.RemoveDeviceRequest) (*pb.RemoveDeviceResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	owner := request.UserId
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		uid, err := grpc.ParseOwnerFromJwtToken(s.ownerClaim, token)
		if err == nil {
			owner = uid
		}
	}

	if owner == "" {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot remove device: invalid UserId"))
	}

	if request.DeviceId == "" {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot remove device: invalid DeviceId"))
	}

	// TODO validate jwt token -> only jwt token is supported

	_, ok, err := tx.Retrieve(request.DeviceId, owner)
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.Internal, "cannot remove device: %v", err.Error()))
	}
	if !ok {
		return nil, log.LogAndReturnError(status.Errorf(codes.NotFound, "cannot remove device: not found"))
	}

	err = tx.Delete(request.DeviceId, owner)
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.NotFound, "cannot remove device: not found"))
	}

	return &pb.RemoveDeviceResponse{}, nil
}
