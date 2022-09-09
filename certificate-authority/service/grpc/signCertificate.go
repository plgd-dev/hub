package grpc

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/pkg/security/certificateSigner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *CertificateAuthorityServer) validateRequest(csr []byte) error {
	infoData, err := getInfoData(csr)
	if err != nil {
		return err
	}
	if infoData.CertificateCommonNameID == s.signerConfig.HubID {
		return fmt.Errorf("common name contains same value as hub id(%v)", s.signerConfig.HubID)
	}
	return nil
}

func (s *CertificateAuthorityServer) SignCertificate(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
	err := s.validateRequest(req.GetCertificateSigningRequest())
	logger := s.logger.With("csr", string(req.GetCertificateSigningRequest()))
	if err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign certificate: %v", err))
	}

	notBefore := s.validFrom()
	notAfter := notBefore.Add(s.validFor)
	signer := certificateSigner.New(s.certificate, s.privateKey, certificateSigner.WithNotBefore(notBefore), certificateSigner.WithNotAfter(notAfter), certificateSigner.WithOverrideCertTemplate(func(template *x509.Certificate) error {
		subject, err := overrideSubject(ctx, template.Subject, s.ownerClaim, s.signerConfig.HubID, "")
		if err != nil {
			return err
		}
		template.Subject = subject
		return nil
	}))
	cert, err := signer.Sign(ctx, req.CertificateSigningRequest)
	if err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign certificate: %v", err))
	}
	logger.With("crt", string(cert)).Debugf("CertificateAuthorityServer.SignCertificate")

	return &pb.SignCertificateResponse{
		Certificate: cert,
	}, nil
}
