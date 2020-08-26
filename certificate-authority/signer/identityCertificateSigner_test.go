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

func newIdentitySigner(t *testing.T) *IdentityCertificateSigner {
	identityIntermediateCABlock, _ := pem.Decode(IdentityIntermediateCA)
	require.NotEmpty(t, identityIntermediateCABlock)
	identityIntermediateCA, err := x509.ParseCertificates(identityIntermediateCABlock.Bytes)
	require.NoError(t, err)
	identityIntermediateCAKeyBlock, _ := pem.Decode(IdentityIntermediateCAKey)
	require.NotEmpty(t, identityIntermediateCAKeyBlock)
	identityIntermediateCAKey, err := x509.ParseECPrivateKey(identityIntermediateCAKeyBlock.Bytes)
	require.NoError(t, err)
	return NewIdentityCertificateSigner(identityIntermediateCA, identityIntermediateCAKey, time.Now(), time.Now().Add(time.Hour*86400))
}

func TestIdentityCertificateSigner_Sign(t *testing.T) {
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
						x509.ExtKeyUsageClientAuth,
						x509.ExtKeyUsageServerAuth,
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

	s := newIdentitySigner(t)

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

var (
	IdentityIntermediateCA = []byte(`-----BEGIN CERTIFICATE-----
MIIBczCCARmgAwIBAgIRANntjEpzu9krzL0EG6fcqqgwCgYIKoZIzj0EAwIwETEP
MA0GA1UEAxMGUm9vdENBMCAXDTE5MDcxOTIwMzczOVoYDzIxMTkwNjI1MjAzNzM5
WjAZMRcwFQYDVQQDEw5JbnRlcm1lZGlhdGVDQTBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABKw1/6WHFcWtw67hH5DzoZvHgA0suC6IYLKms4IP/pds9wU320eDaENo
5860TOyKrGn7vW/cj/OVe2Dzr4KSFVijSDBGMA4GA1UdDwEB/wQEAwIBBjATBgNV
HSUEDDAKBggrBgEFBQcDATASBgNVHRMBAf8ECDAGAQH/AgEAMAsGA1UdEQQEMAKC
ADAKBggqhkjOPQQDAgNIADBFAiEAgPtnYpgwxmPhN0Mo8VX582RORnhcdSHMzFjh
P/li1WwCIFVVWBOrfBnTt7A6UfjP3ljAyHrJERlMauQR+tkD/aqm
-----END CERTIFICATE-----
`)
	IdentityIntermediateCAKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIPF4DPvFeiRL1G0ROd6MosoUGnvIG/2YxH0CbHwnLKxqoAoGCCqGSM49
AwEHoUQDQgAErDX/pYcVxa3DruEfkPOhm8eADSy4Lohgsqazgg/+l2z3BTfbR4No
Q2jnzrRM7Iqsafu9b9yP85V7YPOvgpIVWA==
-----END EC PRIVATE KEY-----
`)
	testCSR = []byte(`-----BEGIN CERTIFICATE REQUEST-----
MIIBRjCB7QIBADA0MTIwMAYDVQQDEyl1dWlkOjAwMDAwMDAwLTAwMDAtMDAwMC0w
MDAwLTAwMDAwMDAwMDAwMTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABLiT0onX
Dw9JpJR9L1+SfyvILLZfluLTuxC7DNa0CdAhrGU2f6SCv+7VJQiQ02wlCt4iFCMx
u1XoaoEZuwcGKaSgVzBVBgkqhkiG9w0BCQ4xSDBGMAwGA1UdEwQFMAMBAQAwCwYD
VR0PBAQDAgGIMCkGA1UdJQQiMCAGCCsGAQUFBwMBBggrBgEFBQcDAgYKKwYBBAGC
3nwBBjAKBggqhkjOPQQDAgNIADBFAiAl/msC2XmurMvieTSOGt9aEgjZ197rchKL
IpK9P9vnXgIhAJ64cyN2X2uWu+x4NqpRkcneK0L3o0yOR4+DxF683pQ2
-----END CERTIFICATE REQUEST-----
`)
)
