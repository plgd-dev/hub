package grpc

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/security/certificateSigner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func overrideSubject(ctx context.Context, subject pkix.Name, ownerClaim, hubID, prefixCommonName string) (pkix.Name, error) {
	if subject.CommonName != "" {
		return subject, nil
	}
	// set subject uuid to owner
	owner, err := grpc.OwnerFromTokenMD(ctx, ownerClaim)
	if err != nil {
		return pkix.Name{}, err
	}
	ownerID := events.OwnerToUUID(owner)
	if hubID == ownerID {
		return pkix.Name{}, fmt.Errorf("common name contains same value as hub id(%v)", hubID)
	}

	subject.CommonName = prefixCommonName + ownerID
	return subject, nil
}

func (s *CertificateAuthorityServer) SignIdentityCertificate(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
	logger := s.logger.With("csr", string(req.GetCertificateSigningRequest()))
	if err := s.validateRequest(req.GetCertificateSigningRequest()); err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign identity certificate: %v", err))
	}
	notBefore := s.validFrom()
	notAfter := notBefore.Add(s.validFor)
	signer := certificateSigner.NewIdentityCertificateSigner(s.certificate, s.privateKey, certificateSigner.WithNotBefore(notBefore), certificateSigner.WithNotAfter(notAfter), certificateSigner.WithOverrideCertTemplate(func(template *x509.Certificate) error {
		subject, err := overrideSubject(ctx, template.Subject, s.ownerClaim, s.signerConfig.HubID, "uuid:")
		if err != nil {
			return err
		}
		template.Subject = subject
		return nil
	}))
	cert, err := signer.Sign(ctx, req.CertificateSigningRequest)
	if err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign identity certificate: %v", err))
	}
	logger.With("crt", string(cert)).Debugf("CertificateAuthorityServer.SignIdentityCertificate")

	return &pb.SignCertificateResponse{
		Certificate: cert,
	}, nil
}
