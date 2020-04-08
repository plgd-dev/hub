package service

import (
	"context"
	"time"

	"github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/authorization/persistence"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AddDevice adds a device to user. It is used by cloud2cloud connector.
func (s *Service) AddDevice(ctx context.Context, request *pb.AddDeviceRequest) (*pb.AddDeviceResponse, error) {
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
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot add device: invalid UserId"))
	}

	if request.DeviceId == "" {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot add device: invalid DeviceId"))
	}

	// TODO validate jwt token -> only jwt token is supported

	dev, ok, err := tx.RetrieveByDevice(request.DeviceId)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, "cannot add device: %v", err.Error()))
	}
	if ok {
		if dev.UserID == userID {
			return &pb.AddDeviceResponse{}, nil
		}
		return nil, logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot add device: devices is owned by another user '%+v'", dev))
	}

	d := persistence.AuthorizedDevice{
		DeviceID:     request.DeviceId,
		UserID:       userID,
		AccessToken:  "",
		RefreshToken: "",
		Expiry:       time.Time{},
	}

	if err := tx.Persist(&d); err != nil {
		return nil, logAndReturnError(status.Errorf(codes.Internal, "cannot add device up: %v", err.Error()))
	}

	return &pb.AddDeviceResponse{}, nil
}
