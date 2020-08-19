package service

import (
	"context"

	"github.com/plgd-dev/cloud/authorization/pb"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

// SignOut verifies device's AccessToken and Expiry required for signing out.
func (s *Service) SignOut(ctx context.Context, request *pb.SignOutRequest) (*pb.SignOutResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	_, err := checkReq(tx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Unauthenticated, "cannot sign out: %v", err))
	}
	return &pb.SignOutResponse{}, nil
}
