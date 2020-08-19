package local

import (
	"context"

	"github.com/plgd-dev/cloud/certificate-authority/pb"
)

type IdentityCertificateSigner struct {
	client pb.CertificateAuthorityClient
}

// NewIdentityCertificateSigner creates an instance.
func NewIdentityCertificateSigner(client pb.CertificateAuthorityClient) *IdentityCertificateSigner {
	return &IdentityCertificateSigner{client: client}
}

// Sign a certificate. A valid access token might be required in the context.
func (s *IdentityCertificateSigner) Sign(ctx context.Context, csr []byte) (signedCsr []byte, err error) {
	req := pb.SignCertificateRequest{CertificateSigningRequest: csr}
	resp, err := s.client.SignIdentityCertificate(ctx, &req)
	return resp.GetCertificate(), err
}
