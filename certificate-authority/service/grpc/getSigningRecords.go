package grpc

import (
	"context"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *CertificateAuthorityServer) GetSigningRecords(req *pb.GetSigningRecordsRequest, srv pb.CertificateAuthority_GetSigningRecordsServer) error {
	owner, err := ownerToUUID(srv.Context(), s.ownerClaim)
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot get signing records: %v", err))
	}
	err = s.store.LoadSigningRecords(srv.Context(), owner, req, func(ctx context.Context, iter store.SigningRecordIter) (err error) {
		for {
			var sub pb.SigningRecord
			if ok := iter.Next(ctx, &sub); !ok {
				return iter.Err()
			}
			if err = srv.Send(&sub); err != nil {
				return err
			}
		}
	})
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot get signing records: %v", err))
	}
	return nil
}
