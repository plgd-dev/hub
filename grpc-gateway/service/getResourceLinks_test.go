package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	oauthTest "github.com/plgd-dev/cloud/oauth-server/test"
	test "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

func TestRequestHandler_GetResourceLinks(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.GetResourceLinksRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.ResourceLink
	}{
		{
			name: "valid",
			args: args{
				req: &pb.GetResourceLinksRequest{},
			},
			wantErr: false,
			want:    test.SortResources(test.ResourceLinksToPb(deviceID, test.GetAllBackendResourceLinks())),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetResourceLinks(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				links := make([]*pb.ResourceLink, 0, 1)
				for {
					link, err := client.Recv()
					if err == io.EOF {
						break
					}
					require.NoError(t, err)
					links = append(links, link)
				}
				test.CheckProtobufs(t, tt.want, test.SortResources(links), test.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
