package service_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

func TestRequestHandler_GetClientConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		want    *pb.ClientConfigurationResponse
	}{
		{
			name: "valid",
			want: &pb.ClientConfigurationResponse{},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxWithoutToken := context.Background()
			got, err := c.GetClientConfiguration(ctxWithoutToken, &pb.ClientConfigurationRequest{})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				test.CheckProtobufs(t, tt.want, got, test.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
