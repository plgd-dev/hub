package service

import (
	"context"

	"github.com/plgd-dev/hub/certificate-authority/pb"
	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/kit/v2/security/signer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) SignIdentityCertificate(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
	err := r.validateRequest(ctx, req.GetCertificateSigningRequest())
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign identity certificate: %v", err))
	}
	notBefore := r.ValidFrom()
	notAfter := notBefore.Add(r.ValidFor)
	signer := signer.NewIdentityCertificateSigner(r.Certificate, r.PrivateKey, notBefore, notAfter)
	cert, err := signer.Sign(ctx, req.CertificateSigningRequest)
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign identity certificate: %v", err))
	}
	log.Debugf("RequestHandler.SignIdentityCertificate csr=%v crt=%v", string(req.CertificateSigningRequest), string(cert))

	return &pb.SignCertificateResponse{
		Certificate: cert,
	}, nil
}
