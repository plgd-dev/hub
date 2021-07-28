package service

import (
	"github.com/plgd-dev/cloud/authorization/persistence"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type request interface {
	GetUserId() string
	GetAccessToken() string
	GetDeviceId() string
}

func checkReq(tx persistence.PersistenceTx, request request) (validUntil int64, err error) {
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
	validUntil, ok = ValidUntil(d.Expiry)
	if !ok {
		return -1, status.Errorf(codes.Unauthenticated, "expired access token")
	}

	return validUntil, nil
}
