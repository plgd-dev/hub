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

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	caTest "github.com/plgd-dev/hub/v2/certificate-authority/test"
	m2mOauthTest "github.com/plgd-dev/hub/v2/m2m-oauth-server/test"
	m2mOauthUri "github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
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
	ctx = pkgGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

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

func createCSRWithKey(t *testing.T, commonName string, priv *ecdsa.PrivateKey) []byte {
	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = commonName
	csr, err := generateCertificate.GenerateCSR(cfg, priv)
	require.NoError(t, err)
	return csr
}

func createCSR(t *testing.T, commonName string) []byte {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return createCSRWithKey(t, commonName, priv)
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

	m2mCfg := m2mOauthTest.MakeConfig(t)
	serviceOAuthClient := m2mOauthTest.ServiceOAuthClient
	serviceOAuthClient.InsertTokenClaims = map[string]interface{}{
		config.OWNER_CLAIM: oauthService.DeviceUserID,
	}
	m2mCfg.OAuthSigner.Clients[0] = &serviceOAuthClient

	tearDown := service.SetUp(ctx, t, service.WithCAConfig(cfg), service.WithM2MOAuthConfig(m2mCfg))
	defer tearDown()

	ctx = pkgGrpc.CtxWithToken(ctx, m2mOauthTest.GetDefaultAccessToken(t))

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

func TestCertificateAuthorityServerSignCSRWithSameDevice(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	cfg := caTest.MakeConfig(t)
	cfg.APIs.GRPC.Authorization.Endpoints = append(cfg.APIs.GRPC.Authorization.Endpoints, validator.AuthorityConfig{
		Authority: "https://" + config.M2M_OAUTH_SERVER_HTTP_HOST + m2mOauthUri.Base,
		HTTP:      config.MakeHttpClientConfig(),
	})

	m2mCfg := m2mOauthTest.MakeConfig(t)
	serviceOAuthClient := m2mOauthTest.ServiceOAuthClient
	serviceOAuthClient.InsertTokenClaims = map[string]interface{}{
		config.OWNER_CLAIM: oauthService.DeviceUserID,
	}
	m2mCfg.OAuthSigner.Clients[0] = &serviceOAuthClient

	tearDown := service.SetUp(ctx, t, service.WithCAConfig(cfg), service.WithM2MOAuthConfig(m2mCfg))
	defer tearDown()

	ctx = pkgGrpc.CtxWithToken(ctx, m2mOauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.CERTIFICATE_AUTHORITY_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewCertificateAuthorityClient(conn)

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	deviceID := uuid.NewString()

	csr := createCSRWithKey(t, "uuid:"+deviceID, priv)
	_, err = c.SignIdentityCertificate(ctx, &pb.SignCertificateRequest{CertificateSigningRequest: csr})
	require.NoError(t, err)

	csr1 := createCSRWithKey(t, "uuid:"+deviceID, priv)
	_, err = c.SignIdentityCertificate(ctx, &pb.SignCertificateRequest{CertificateSigningRequest: csr1})
	require.NoError(t, err)

	priv2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	csr2 := createCSRWithKey(t, "uuid:"+deviceID, priv2)
	_, err = c.SignIdentityCertificate(ctx, &pb.SignCertificateRequest{CertificateSigningRequest: csr2})
	require.Error(t, err)
}

func TestCertificateAuthorityServerSignCSRWithEmptyCommonName(t *testing.T) {
	csr := createCSR(t, "")
	testSigningByFunction(t, func(ctx context.Context, c pb.CertificateAuthorityClient, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		return c.SignCertificate(ctx, req)
	}, csr)
}
