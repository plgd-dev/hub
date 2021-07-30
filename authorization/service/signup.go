package service

import (
	"context"
	"strings"

	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/authorization/persistence"
	"github.com/plgd-dev/cloud/pkg/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SignUp exchanges Auth Code for Access Token via OAuth.
// The Access Token can be used for signing the device in/out,
// or for authorizing the device to act as the user.
func (s *Service) SignUp(ctx context.Context, request *pb.SignUpRequest) (*pb.SignUpResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	if request.GetDeviceId() == "" {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign up: invalid DeviceId"))
	}
	if request.GetAuthorizationCode() == "" {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign up: invalid AuthorizationCode"))
	}

	token, err := s.deviceProvider.Exchange(ctx, request.AuthorizationProvider, request.AuthorizationCode)
	if err != nil {
		code := codes.Unauthenticated
		if strings.Contains(err.Error(), "connect: connection refused") {
			code = codes.Unavailable
		}
		return nil, log.LogAndReturnError(status.Errorf(code, "cannot sign up: %v", err.Error()))
	}

	dev, ok, err := tx.RetrieveByDevice(request.DeviceId)
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.Internal, "cannot sign up: %v", err.Error()))
	}
	if ok {
		if dev.Owner != token.Owner {
			return nil, log.LogAndReturnError(status.Errorf(codes.Unauthenticated, "cannot sign up: devices is owned by another user"))
		}
	}

	d := persistence.AuthorizedDevice{
		DeviceID:     request.DeviceId,
		Owner:        token.Owner,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	if err := tx.Persist(&d); err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.Internal, "cannot sign up: %v", err.Error()))
	}

	validUntil, ok := ValidUntil(token.Expiry)
	if !ok {
		return nil, log.LogAndReturnError(status.Errorf(codes.Unauthenticated, "cannot sign up: expired access token"))
	}

	return &pb.SignUpResponse{
		AccessToken:  token.AccessToken,
		UserId:       token.Owner,
		RefreshToken: token.RefreshToken,
		ValidUntil:   validUntil,
		RedirectUri:  "",
	}, nil
}
