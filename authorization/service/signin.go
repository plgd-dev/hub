package service

import (
	"context"

	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/authorization/persistence"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type request interface {
	GetUserId() string
	GetAccessToken() string
	GetDeviceId() string
}

func checkReq(tx persistence.PersistenceTx, request request) (expiresInSeconds int64, err error) {
	if request.GetUserId() == "" {
		return -1, status.Errorf(codes.InvalidArgument, "invalid UserId")
	}
	if request.GetAccessToken() == "" {
		return -1, status.Errorf(codes.InvalidArgument, "invalid AccessToken")
	}
	if request.GetDeviceId() == "" {
		return -1, status.Errorf(codes.InvalidArgument, "invalid DeviceId")
	}

	d, ok, err := tx.Retrieve(request.GetDeviceId(), request.GetUserId())
	if err != nil {
		return -1, status.Errorf(codes.Internal, err.Error())
	}
	if !ok {
		return -1, status.Errorf(codes.Unauthenticated, "not found")
	}
	if d.AccessToken != request.GetAccessToken() {
		return -1, status.Errorf(codes.Unauthenticated, "bad AccessToken")
	}
	if d.Owner != request.GetUserId() {
		return -1, status.Errorf(codes.Unauthenticated, "bad UserId")
	}
	expiresIn, ok := ExpiresIn(d.Expiry)
	if !ok {
		return -1, status.Errorf(codes.Unauthenticated, "expired access token")
	}

	return expiresIn, nil
}

// SignIn verifies device's AccessToken and Expiry required for signing in.
func (s *Service) SignIn(ctx context.Context, request *pb.SignInRequest) (*pb.SignInResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	expiresIn, err := checkReq(tx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Unauthenticated, "cannot sign in: %v", err))
	}

	return &pb.SignInResponse{ExpiresIn: expiresIn}, nil
}
