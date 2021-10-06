package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/v2/certificate-authority/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"github.com/plgd-dev/cloud/v2/test"
	testCfg "github.com/plgd-dev/cloud/v2/test/config"
	oauthTest "github.com/plgd-dev/cloud/v2/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type ClientSignFunc = func(context.Context, pb.CertificateAuthorityClient, *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error)

func testSigningByFunction(t *testing.T, signFn ClientSignFunc) {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	conn, err := grpc.Dial(testCfg.CERTIFICATE_AUTHORITY_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewCertificateAuthorityClient(conn)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := signFn(ctx, c, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}

func TestRequestHandlerSignCertificate(t *testing.T) {
	testSigningByFunction(t, func(ctx context.Context, c pb.CertificateAuthorityClient, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		return c.SignCertificate(ctx, req)
	})
}

var (
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
