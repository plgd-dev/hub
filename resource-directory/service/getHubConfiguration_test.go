package service_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	rdTest "github.com/plgd-dev/hub/v2/resource-directory/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerGetHubConfiguration(t *testing.T) {
	expected := rdTest.MakeConfig(t).ExposedHubConfiguration.ToProto(config.HubID())
	expected.CurrentTime = 0
	tests := []struct {
		name    string
		wantErr bool
		want    *pb.HubConfigurationResponse
	}{
		{
			name: "valid",
			want: expected,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxWithoutToken := context.Background()
			got, err := c.GetHubConfiguration(ctxWithoutToken, &pb.HubConfigurationRequest{})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, got.GetCertificateAuthorities())
				got.CertificateAuthorities = ""
				require.NotEqual(t, int64(0), got.GetCurrentTime())
				got.CurrentTime = 0
				test.CheckProtobufs(t, tt.want, got, test.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
