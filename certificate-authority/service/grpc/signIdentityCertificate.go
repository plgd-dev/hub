package grpc

import (
	"context"
	"crypto/x509/pkix"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ownerToUUID(ctx context.Context, ownerClaim string) (string, error) {
	owner, err := grpc.OwnerFromTokenMD(ctx, ownerClaim)
	if err != nil {
		return "", err
	}
	return events.OwnerToUUID(owner), nil
}

func overrideSubject(ctx context.Context, subject pkix.Name, ownerClaim, hubID, prefixCommonName string) (pkix.Name, error) {
	if subject.CommonName != "" {
		return subject, nil
	}
	// set subject uuid to owner
	ownerID, err := ownerToUUID(ctx, ownerClaim)
	if err != nil {
		return pkix.Name{}, err
	}
	if hubID == ownerID {
		return pkix.Name{}, fmt.Errorf("common name contains same value as hub id(%v)", hubID)
	}

	subject.CommonName = prefixCommonName + ownerID
	return subject, nil
}

func (s *CertificateAuthorityServer) SignIdentityCertificate(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
	const fmtError = "cannot sign identity certificate: %v"
	logger := s.logger.With("csr", string(req.GetCertificateSigningRequest()))
	if err := s.validateRequest(req.GetCertificateSigningRequest()); err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, err))
	}
	signer := s.GetSigner()
	if signer == nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, errors.New("signer is empty")))
	}
	cert, signingRecord, err := signer.SignIdentityCSR(ctx, req.GetCertificateSigningRequest())
	if err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, err))
	}
	if signingRecord.GetCredential() == nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, "cannot create signing record"))
	}
	signingRecord.Credential.CertificatePem = string(cert)
	if err := s.updateSigningRecord(ctx, signingRecord); err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, fmtError, err))
	}
	logger.With("crt", string(cert)).Debugf("CertificateAuthorityServer.SignIdentityCertificate")

	return &pb.SignCertificateResponse{
		Certificate: cert,
	}, nil
}
