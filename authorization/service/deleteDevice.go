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

// DeleteDevice removes a device from user. It is used by cloud2cloud connector.
func (s *Service) DeleteDevice(ctx context.Context, request *pb.DeleteDeviceRequest) (*pb.DeleteDeviceResponse, error) {
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
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot delete device: invalid UserId"))
	}

	if request.DeviceId == "" {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot delete device: invalid DeviceId"))
	}

	// TODO validate jwt token -> only jwt token is supported

	_, ok, err := tx.Retrieve(request.DeviceId, owner)
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.Internal, "cannot delete device: %v", err.Error()))
	}
	if !ok {
		return nil, log.LogAndReturnError(status.Errorf(codes.NotFound, "cannot delete device: not found"))
	}

	err = tx.Delete(request.DeviceId, owner)
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.NotFound, "cannot delete device: not found"))
	}

	return &pb.DeleteDeviceResponse{}, nil
}
