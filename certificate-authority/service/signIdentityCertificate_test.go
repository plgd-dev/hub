package service

import (
	"context"
	"testing"

	"github.com/go-ocf/cloud/certificate-authority/pb"
	"github.com/stretchr/testify/require"
)

func TestRequestHandler_SignIdentityCertificate(t *testing.T) {
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
			got, err := r.SignIdentityCertificate(context.Background(), tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
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
