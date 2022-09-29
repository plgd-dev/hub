package grpc_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/security/generateCertificate"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type ClientSignFunc = func(context.Context, pb.CertificateAuthorityClient, *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error)

func testSigningByFunction(t *testing.T, signFn ClientSignFunc, csr []byte) {
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
			name: "invalid csr",
			args: args{
				req: &pb.SignCertificateRequest{},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				req: &pb.SignCertificateRequest{
					CertificateSigningRequest: csr,
				},
			},
			wantErr: false,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

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

func TestCertificateAuthorityServerSignCSR(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = "aa"
	csr, err := generateCertificate.GenerateCSR(cfg, priv)
	require.NoError(t, err)
	testSigningByFunction(t, func(ctx context.Context, c pb.CertificateAuthorityClient, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		return c.SignCertificate(ctx, req)
	}, csr)
}

func TestCertificateAuthorityServerSignCSRWithEmptyCommonName(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	var cfg generateCertificate.Configuration
	csr, err := generateCertificate.GenerateCSR(cfg, priv)
	require.NoError(t, err)
	testSigningByFunction(t, func(ctx context.Context, c pb.CertificateAuthorityClient, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		return c.SignCertificate(ctx, req)
	}, csr)
}
