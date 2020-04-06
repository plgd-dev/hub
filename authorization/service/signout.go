package service

import (
	"context"

	"github.com/go-ocf/ocf-cloud/authorization/pb"
	"github.com/go-ocf/kit/log"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SignOut verifies device's AccessToken and Expiry required for signing out.
func (s *Service) SignOut(ctx context.Context, request *pb.SignOutRequest) (*pb.SignOutResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign out: %v", err))
	}

	userID, err := parseSubFromJwtToken(token)
	if err != nil {
		log.Debugf("cannot parse user from jwt token: %v", err)
		userID = request.UserId
	}

	if userID == "" {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign out: invalid UserId"))
	}

	d, ok, err := tx.Retrieve(request.DeviceId, userID)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, "cannot sign out: %v", err.Error()))
	}
	if !ok {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign out: not found"))
	}
	if d.AccessToken != token {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign out: unexpected access token"))
	}

	_, ok = ExpiresIn(d.Expiry)
	if !ok {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot sign out: expired access token"))
	}
	return &pb.SignOutResponse{}, nil
}
