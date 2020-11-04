package service

import (
	"context"
	"time"

	"github.com/karrick/tparse/v2"
	"github.com/plgd-dev/cloud/certificate-authority/pb"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/signer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) SignCertificate(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
	log.Debugf("RequestHandler.SignCertificate: %v", string(req.CertificateSigningRequest))

	notBefore, err := tparse.ParseNow(time.RFC3339, r.ValidFrom)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "invalid VALID_FROM(%v): %v", r.ValidFrom, err))
	}

	notAfter := notBefore.Add(r.ValidFor)
	signer := signer.NewBasicCertificateSigner(r.Certificate, r.PrivateKey, notBefore, notAfter)
	cert, err := signer.Sign(ctx, req.CertificateSigningRequest)
	if err != nil {
		return nil, logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot sign certificate: %v", err))
	}
	return &pb.SignCertificateResponse{
		Certificate: cert,
	}, nil
}
