package service

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func checkReq(request CoapSignInReq) error {
	if request.UserID == "" {
		return status.Errorf(codes.InvalidArgument, "invalid UserId")
	}
	if request.AccessToken == "" {
		return status.Errorf(codes.InvalidArgument, "invalid AccessToken")
	}
	return nil
}
