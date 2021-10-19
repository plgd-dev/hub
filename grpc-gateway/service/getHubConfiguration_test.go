package service_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	rdTest "github.com/plgd-dev/hub/resource-directory/test"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	pbTest "github.com/plgd-dev/hub/test/pb"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandler_GetHubConfiguration(t *testing.T) {
	expected := rdTest.MakeConfig(t).ExposedHubConfiguration.ToProto()
	expected.CurrentTime = 0
	tests := []struct {
		name string
		want *pb.HubConfigurationResponse
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

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxWithoutToken := context.Background()
			got, err := c.GetHubConfiguration(ctxWithoutToken, &pb.HubConfigurationRequest{})
			require.NoError(t, err)
			pbTest.CmpHubConfigurationResponse(t, tt.want, got)
		})
	}
}
