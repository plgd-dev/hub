package grpc

import (
	"context"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func errDeleteSigningRecords(err error) error {
	return status.Errorf(codes.InvalidArgument, "cannot delete signing records: %v", err)
}

func (s *CertificateAuthorityServer) DeleteSigningRecords(ctx context.Context, req *pb.DeleteSigningRecordsRequest) (*pb.DeletedSigningRecords, error) {
	owner, err := ownerToUUID(ctx, s.ownerClaim)
	if err != nil {
		return nil, s.logger.LogAndReturnError(errDeleteSigningRecords(err))
	}
	count, err := s.store.RevokeSigningRecords(ctx, owner, req)
	if err != nil {
		return nil, s.logger.LogAndReturnError(errDeleteSigningRecords(err))
	}
	return &pb.DeletedSigningRecords{
		Count: count,
	}, nil
}
