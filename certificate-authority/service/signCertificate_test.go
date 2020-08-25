package service

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/go-ocf/cloud/certificate-authority/pb"
	"github.com/stretchr/testify/require"
)

func newRequestHandler(t *testing.T) *RequestHandler {
	identityIntermediateCABlock, _ := pem.Decode(IdentityIntermediateCA)
	require.NotEmpty(t, identityIntermediateCABlock)
	identityIntermediateCA, err := x509.ParseCertificates(identityIntermediateCABlock.Bytes)
	require.NoError(t, err)
	identityIntermediateCAKeyBlock, _ := pem.Decode(IdentityIntermediateCAKey)
	require.NotEmpty(t, identityIntermediateCAKeyBlock)
	identityIntermediateCAKey, err := x509.ParseECPrivateKey(identityIntermediateCAKeyBlock.Bytes)
	require.NoError(t, err)
	return &RequestHandler{
		ValidFrom: func() time.Time {
			return time.Now()
		},
		ValidFor:    time.Hour * 86400,
		Certificate: identityIntermediateCA,
		PrivateKey:  identityIntermediateCAKey,
	}
}

func TestRequestHandler_SignCertificate(t *testing.T) {
	type args struct {
		req *pb.SignCertificateRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.SignCertificateResponse
		wantErr bool
	}{
		{
			name: "invalid auth",
			args: args{
				req: &pb.SignCertificateRequest{},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				req: &pb.SignCertificateRequest{
					CertificateSigningRequest: testCSR,
				},
			},
			wantErr: false,
		},
	}

	r := newRequestHandler(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.SignCertificate(context.Background(), tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}
