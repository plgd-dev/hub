package service

import (
	"context"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"

	"github.com/go-ocf/ocf-cloud/authorization/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RemoveDevice remove a device from user. It is used by openapi connector.
func (s *Service) RemoveDevice(ctx context.Context, request *pb.RemoveDeviceRequest) (*pb.RemoveDeviceResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	userID := request.UserId
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		uid, err := parseSubFromJwtToken(token)
		if err == nil {
			userID = uid
		}
	}

	if userID == "" {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot remove device: invalid UserId"))
	}

	if request.DeviceId == "" {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot remove device: invalid DeviceId"))
	}

	// TODO validate jwt token -> only jwt token is supported

	_, ok, err := tx.Retrieve(request.DeviceId, userID)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, "cannot remove device: %v", err.Error()))
	}
	if !ok {
		return nil, logAndReturnError(status.Errorf(codes.NotFound, "cannot remove device: not found"))
	}

	err = tx.Delete(request.DeviceId, userID)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.NotFound, "cannot remove device: not found"))
	}

	return &pb.RemoveDeviceResponse{}, nil
}
