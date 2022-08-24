package service

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/pkg/security/certificateSigner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) validateRequest(csr []byte) error {
	infoData, err := getInfoData(csr)
	if err != nil {
		return err
	}
	if infoData.CertificateCommonNameID == r.Config.Signer.HubID {
		return fmt.Errorf("common name contains same value as hub id(%v)", r.Config.Signer.HubID)
	}
	return nil
}

func (r *RequestHandler) SignCertificate(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
	err := r.validateRequest(req.GetCertificateSigningRequest())
	logger := r.logger.With("csr", string(req.GetCertificateSigningRequest()))
	if err != nil {
		return nil, logger.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign certificate: %v", err))
	}

	notBefore := r.ValidFrom()
	notAfter := notBefore.Add(r.ValidFor)
	signer := certificateSigner.New(r.Certificate, r.PrivateKey, certificateSigner.WithNotBefore(notBefore), certificateSigner.WithNotAfter(notAfter), certificateSigner.WithOverrideCertTemplate(func(template *x509.Certificate) error {
		subject, err := overrideSubject(ctx, template.Subject, r.Config.APIs.GRPC.Authorization.OwnerClaim, r.Config.Signer.HubID, "")
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
	logger.With("crt", string(cert)).Debugf("RequestHandler.SignCertificate")

	return &pb.SignCertificateResponse{
		Certificate: cert,
	}, nil
}
