package signer

import (
	"context"

	"github.com/plgd-dev/hub/certificate-authority/pb"
)

type BasicCertificateSigner struct {
	client pb.CertificateAuthorityClient
}

// NewBasicCertificateSigner creates an instance.
func NewBasicCertificateSigner(client pb.CertificateAuthorityClient) *BasicCertificateSigner {
	return &BasicCertificateSigner{client: client}
}

// Sign a certificate. A valid access token might be required in the context.
func (s *BasicCertificateSigner) Sign(ctx context.Context, csr []byte) (signedCsr []byte, err error) {
	req := pb.SignCertificateRequest{CertificateSigningRequest: csr}
	resp, err := s.client.SignCertificate(ctx, &req)
	return resp.GetCertificate(), err
}
