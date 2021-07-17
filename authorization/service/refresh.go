package service

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/authorization/persistence"
)

// RefreshToken renews AccessToken using RefreshToken.
func (s *Service) RefreshToken(ctx context.Context, request *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	token, err := s.deviceProvider.Refresh(ctx, request.RefreshToken)
	if err != nil {
		code := codes.Unauthenticated
		if strings.Contains(err.Error(), "connect: connection refused") {
			code = codes.Unavailable
		}
		return nil, logAndReturnError(status.Errorf(code, "cannot refresh token: %v", err))
	}

	owner := token.Owner
	if owner == "" {
		owner = request.UserId
	}
	if owner == "" {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot refresh token: cannot determine owner"))
	}

	d := persistence.AuthorizedDevice{
		DeviceID:     request.DeviceId,
		Owner:        owner,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	if err := tx.Persist(&d); err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, "cannot refresh token: err"))
	}

	validUntil, ok := ValidUntil(token.Expiry)
	if !ok {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot refresh token: expired access token"))
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ValidUntil:   validUntil,
	}, nil
}
