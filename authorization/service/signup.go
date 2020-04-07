package service

import (
	"context"

	"github.com/go-ocf/kit/log"

	"github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/authorization/persistence"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SignUp exchanges Auth Code for Access Token via OAuth.
// The Access Token can be used for signing the device in/out,
// or for authorizing the device to act as the user.
func (s *Service) SignUp(ctx context.Context, request *pb.SignUpRequest) (*pb.SignUpResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	log.Debugf("Service.SignUp \"%v\"", request.AuthorizationCode)

	token, err := s.deviceProvider.Exchange(ctx, request.AuthorizationProvider, request.AuthorizationCode)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot sign up: %v", err.Error()))
	}

	dev, ok, err := tx.RetrieveByDevice(request.DeviceId)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, "cannot sign up: %v", err.Error()))
	}
	if ok {
		if dev.UserID != token.UserID {
			return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot sign up: devices is owned by another user"))
		}
	}

	d := persistence.AuthorizedDevice{
		DeviceID:     request.DeviceId,
		UserID:       token.UserID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	if err := tx.Persist(&d); err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, "cannot sign up: %v", err.Error()))
	}

	expiresIn, ok := ExpiresIn(token.Expiry)
	if !ok {
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot sign up: expired access token"))
	}

	return &pb.SignUpResponse{
		AccessToken:  token.AccessToken,
		UserId:       token.UserID,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    expiresIn,
		RedirectUri:  "",
	}, nil
}
