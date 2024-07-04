package grpc_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"strconv"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	caTest "github.com/plgd-dev/hub/v2/certificate-authority/test"
	m2mOauthTest "github.com/plgd-dev/hub/v2/m2m-oauth-server/test"
	m2mOauthUri "github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type ClientSignFunc = func(context.Context, pb.CertificateAuthorityClient, *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error)

func testSigningByFunction(t *testing.T, signFn ClientSignFunc, csr ...[]byte) {
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
	}
	for idx, csr := range csr {
		tests = append(tests, struct {
			name    string
			args    args
			want    *pb.SignCertificateResponse
			wantErr bool
		}{
			name: "valid-" + strconv.Itoa(idx),
			args: args{
				req: &pb.SignCertificateRequest{
					CertificateSigningRequest: csr,
				},
			},
			wantErr: false,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.CERTIFICATE_AUTHORITY_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
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

func createCSR(t *testing.T, commonName string) []byte {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = commonName
	csr, err := generateCertificate.GenerateCSR(cfg, priv)
	require.NoError(t, err)
	return csr
}

func TestCertificateAuthorityServerSignCSR(t *testing.T) {
	csr := createCSR(t, "aa")
	testSigningByFunction(t, func(ctx context.Context, c pb.CertificateAuthorityClient, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		return c.SignCertificate(ctx, req)
	}, csr, csr)
}

func TestCertificateAuthorityServerSignCSRWithDifferentPublicKeys(t *testing.T) {
	csr := createCSR(t, "bb")
	csr1 := createCSR(t, "bb")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	cfg := caTest.MakeConfig(t)
	cfg.APIs.GRPC.Authorization.Endpoints = append(cfg.APIs.GRPC.Authorization.Endpoints, validator.AuthorityConfig{
		Authority: "https://" + config.M2M_OAUTH_SERVER_HTTP_HOST + m2mOauthUri.Base,
		HTTP:      config.MakeHttpClientConfig(),
	})

	tearDown := service.SetUp(ctx, t, service.WithCAConfig(cfg))
	defer tearDown()

	ctx = kitNetGrpc.CtxWithToken(ctx, m2mOauthTest.GetDefaultAccessToken(t, m2mOauthTest.WithAccessTokenOwner(oauthService.DeviceUserID)))

	conn, err := grpc.NewClient(config.CERTIFICATE_AUTHORITY_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewCertificateAuthorityClient(conn)

	_, err = c.SignIdentityCertificate(ctx, &pb.SignCertificateRequest{CertificateSigningRequest: csr})
	require.NoError(t, err)

	_, err = c.SignIdentityCertificate(ctx, &pb.SignCertificateRequest{CertificateSigningRequest: csr1})
	require.NoError(t, err)
}

func TestCertificateAuthorityServerSignCSRWithEmptyCommonName(t *testing.T) {
	csr := createCSR(t, "")
	testSigningByFunction(t, func(ctx context.Context, c pb.CertificateAuthorityClient, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		return c.SignCertificate(ctx, req)
	}, csr)
}
