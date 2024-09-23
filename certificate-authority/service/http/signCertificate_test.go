package http_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	certAuthURI "github.com/plgd-dev/hub/v2/certificate-authority/service/uri"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type ClientSignFunc = func(context.Context, *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error)

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
		},
		{
			name: "valid - new with the same csr",
			args: args{
				req: &pb.SignCertificateRequest{
					CertificateSigningRequest: csr,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = pkgGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := signFn(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}

func httpDoSign(ctx context.Context, t *testing.T, uri string, req *pb.SignCertificateRequest, resp *pb.SignCertificateResponse) error {
	token, err := pkgGrpc.TokenFromOutgoingMD(ctx)
	require.NoError(t, err)
	reqBody, err := protojson.Marshal(req)
	require.NoError(t, err)
	request := httpgwTest.NewRequest(http.MethodPost, uri, io.NopCloser(bytes.NewReader(reqBody))).Host(config.CERTIFICATE_AUTHORITY_HTTP_HOST).AuthToken(token).Build()
	httpResp := httpgwTest.HTTPDo(t, request)
	respBody, err := io.ReadAll(httpResp.Body)
	require.NoError(t, err)
	err = httpResp.Body.Close()
	require.NoError(t, err)
	if httpResp.StatusCode != http.StatusOK {
		var errStatus spb.Status
		err = protojson.Unmarshal(respBody, &errStatus)
		require.NoError(t, err)
		return status.ErrorProto(&errStatus)
	}
	err = protojson.Unmarshal(respBody, resp)
	require.NoError(t, err)
	return nil
}

func TestCertificateAuthorityServerSignCSR(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	var cfg generateCertificate.Configuration
	cfg.Subject.CommonName = "aa"
	csr, err := generateCertificate.GenerateCSR(cfg, priv)
	require.NoError(t, err)
	testSigningByFunction(t, func(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		var resp pb.SignCertificateResponse
		return &resp, httpDoSign(ctx, t, certAuthURI.SignCertificate, req, &resp)
	}, csr)
}

func TestCertificateAuthorityServerSignCSRWithEmptyCommonName(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	var cfg generateCertificate.Configuration
	csr, err := generateCertificate.GenerateCSR(cfg, priv)
	require.NoError(t, err)
	testSigningByFunction(t, func(ctx context.Context, req *pb.SignCertificateRequest) (*pb.SignCertificateResponse, error) {
		var resp pb.SignCertificateResponse
		return &resp, httpDoSign(ctx, t, certAuthURI.SignCertificate, req, &resp)
	}, csr)
}
