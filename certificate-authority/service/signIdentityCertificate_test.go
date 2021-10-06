package service_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/cloud/v2/certificate-authority/pb"
)

func TestRequestHandlerSignIdentityCertificate(t *testing.T) {
	testSigningByFunction(t, func(ctx context.Context, c pb.CertificateAuthorityClient, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		return c.SignIdentityCertificate(ctx, req)
	})
}
