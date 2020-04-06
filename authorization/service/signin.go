package service

import (
	"context"

	"github.com/go-ocf/ocf-cloud/authorization/pb"
	"github.com/go-ocf/kit/log"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SignIn verifies device's AccessToken and Expiry required for signing in.
func (s *Service) SignIn(ctx context.Context, request *pb.SignInRequest) (*pb.SignInResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign in: %v", err))
	}

	userID, err := parseSubFromJwtToken(token)
	if err != nil {
		log.Debugf("cannot parse user from jwt token: %v", err)
		userID = request.UserId
	}

	if userID == "" {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign in: invalid UserId"))
	}

	d, ok, err := tx.Retrieve(request.DeviceId, userID)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, "cannot sign in: %v", err.Error()))
	}
	if !ok {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign in: not found"))
	}

	expiresIn, ok := ExpiresIn(d.Expiry)
	if !ok {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot sign in: expired access token"))
	}

	return &pb.SignInResponse{ExpiresIn: expiresIn}, nil
}
