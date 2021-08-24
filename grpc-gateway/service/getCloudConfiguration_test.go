package service_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	rdTest "github.com/plgd-dev/cloud/resource-directory/test"
	"github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/cloud/test/config"
	testCfg "github.com/plgd-dev/cloud/test/config"
)

func TestRequestHandler_GetCloudConfiguration(t *testing.T) {
	expected := rdTest.MakeConfig(t).ExposedCloudConfiguration.ToProto(config.MakeAuthURL())
	expected.CurrentTime = 0
	tests := []struct {
		name    string
		wantErr bool
		want    *pb.CloudConfigurationResponse
	}{
		{
			name: "valid",
			want: expected,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

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
			got, err := c.GetCloudConfiguration(ctxWithoutToken, &pb.CloudConfigurationRequest{})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, got.CloudCertificateAuthorities)
				got.CloudCertificateAuthorities = ""
				require.NotEqual(t, int64(0), got.CurrentTime)
				got.CurrentTime = 0
				test.CheckProtobufs(t, tt.want, got, test.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
