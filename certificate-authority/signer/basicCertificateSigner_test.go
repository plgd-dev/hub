package signer

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newBasicSigner(t *testing.T) *BasicCertificateSigner {
	identityIntermediateCABlock, _ := pem.Decode(IdentityIntermediateCA)
	require.NotEmpty(t, identityIntermediateCABlock)
	identityIntermediateCA, err := x509.ParseCertificates(identityIntermediateCABlock.Bytes)
	require.NoError(t, err)
	identityIntermediateCAKeyBlock, _ := pem.Decode(IdentityIntermediateCAKey)
	require.NotEmpty(t, identityIntermediateCAKeyBlock)
	identityIntermediateCAKey, err := x509.ParseECPrivateKey(identityIntermediateCAKeyBlock.Bytes)
	require.NoError(t, err)
	return NewBasicCertificateSigner(identityIntermediateCA, identityIntermediateCAKey, time.Now(), time.Now().Add(time.Hour*86400))
}

func TestBasicCertificateSigner_Sign(t *testing.T) {
	type args struct {
		csr []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []*x509.Certificate
		wantErr bool
	}{
		{
			name:    "invalid",
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				csr: testCSR,
			},
			wantErr: false,
			want: []*x509.Certificate{
				{
					Subject: pkix.Name{
						CommonName: "uuid:00000000-0000-0000-0000-000000000001",
					},
					ExtKeyUsage: []x509.ExtKeyUsage{
						x509.ExtKeyUsageServerAuth,
						x509.ExtKeyUsageClientAuth,
					},
					UnknownExtKeyUsage: []asn1.ObjectIdentifier{asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 44924, 1, 6}},
				},
				{
					Subject: pkix.Name{
						CommonName: "IntermediateCA",
					},
					ExtKeyUsage: []x509.ExtKeyUsage{
						x509.ExtKeyUsageServerAuth,
					},
				},
			},
		},
	}

	s := newBasicSigner(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.Sign(context.Background(), tt.args.csr)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			for i := 0; i < len(tt.want); i++ {
				block, rest := pem.Decode(got)
				require.NotEmpty(t, block.Bytes)
				cert, err := x509.ParseCertificate(block.Bytes)
				require.NoError(t, err)
				require.Equal(t, tt.want[i].Subject.CommonName, cert.Subject.CommonName)
				require.Equal(t, tt.want[i].ExtKeyUsage, cert.ExtKeyUsage)
				require.Equal(t, tt.want[i].UnknownExtKeyUsage, cert.UnknownExtKeyUsage)
				got = rest
			}
		})
	}
}
