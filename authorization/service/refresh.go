package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-ocf/ocf-cloud/authorization/pb"
	"github.com/go-ocf/ocf-cloud/authorization/persistence"
)

// RefreshToken renews AccessToken using RefreshToken.
func (s *Service) RefreshToken(ctx context.Context, request *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	token, err := s.deviceProvider.Refresh(ctx, request.RefreshToken)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot refresh token: %v", err))
	}

	userID := token.UserID
	if userID == "" {
		userID = request.UserId
	}
	if userID == "" {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot refresh token: cannot determine userId"))
	}

	d := persistence.AuthorizedDevice{
		DeviceID:     request.DeviceId,
		UserID:       userID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	if err := tx.Persist(&d); err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, "cannot refresh token: err"))
	}

	expiresIn, ok := ExpiresIn(token.Expiry)
	if !ok {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot refresh token: expired access token"))
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}
