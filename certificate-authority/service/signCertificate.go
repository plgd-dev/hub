package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/kit/v2/security/signer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) validateRequest(ctx context.Context, csr []byte) error {
	infoData, err := getInfoData(ctx, csr)
	if err != nil {
		return fmt.Errorf("cannot get info data for csr=%v: %w", string(csr), err)
	}
	if infoData.CertificateCommonNameID == r.Config.Signer.HubID {
		return fmt.Errorf("csr=%v common name contains same value as hub id(%v)", string(csr), r.Config.Signer.HubID)
	}
	return nil
}

func (r *RequestHandler) SignCertificate(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
	err := r.validateRequest(ctx, req.GetCertificateSigningRequest())
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign certificate: %v", err))
	}

	notBefore := r.ValidFrom()
	notAfter := notBefore.Add(r.ValidFor)
	signer := signer.NewBasicCertificateSigner(r.Certificate, r.PrivateKey, notBefore, notAfter)
	cert, err := signer.Sign(ctx, req.CertificateSigningRequest)
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign certificate: %v", err))
	}
	log.Debugf("RequestHandler.SignCertificate csr=%v crt=%v", string(req.CertificateSigningRequest), string(cert))

	return &pb.SignCertificateResponse{
		Certificate: cert,
	}, nil
}
