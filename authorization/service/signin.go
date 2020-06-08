package service

import (
	"context"

	"github.com/go-ocf/cloud/authorization/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SignIn verifies device's AccessToken and Expiry required for signing in.
func (s *Service) SignIn(ctx context.Context, request *pb.SignInRequest) (*pb.SignInResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	if request.GetUserId() == "" {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign in: invalid UserId"))
	}
	if request.GetAccessToken() == "" {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign in: invalid AccessToken"))
	}
	if request.GetDeviceId() == "" {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign in: invalid DeviceId"))
	}

	d, ok, err := tx.Retrieve(request.DeviceId, request.GetUserId())
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, "cannot sign in: %v", err.Error()))
	}
	if !ok {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot sign in: not found"))
	}
	if d.AccessToken != request.GetAccessToken() {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot sign in: bad AccessToken"))
	}

	expiresIn, ok := ExpiresIn(d.Expiry)
	if !ok {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot sign in: expired access token"))
	}

	return &pb.SignInResponse{ExpiresIn: expiresIn}, nil
}
