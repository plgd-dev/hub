package grpc

import (
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *CertificateAuthorityServer) GetSigningRecords(req *pb.GetSigningRecordsRequest, srv pb.CertificateAuthority_GetSigningRecordsServer) error {
	owner, err := ownerToUUID(srv.Context(), s.ownerClaim)
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot get signing records: %v", err))
	}
	err = s.store.LoadSigningRecords(srv.Context(), owner, req, func(sr *pb.SigningRecord) (err error) {
		if err = srv.Send(sr); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return s.logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot get signing records: %v", err))
	}
	return nil
}
