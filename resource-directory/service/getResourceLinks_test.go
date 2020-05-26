package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	grpcTest "github.com/go-ocf/cloud/grpc-gateway/test"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
)

func TestRequestHandler_GetResourceLinks(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
	type args struct {
		req *pb.GetResourceLinksRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []pb.ResourceLink
	}{
		{
			name: "valid",
			args: args{
				req: &pb.GetResourceLinksRequest{},
			},
			wantErr: false,
			want:    grpcTest.SortResources(grpcTest.ResourceLinksToPb(deviceID, grpcTest.GetAllBackendResourceLinks())),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(grpcTest.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: grpcTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c, deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetResourceLinks(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				links := make([]pb.ResourceLink, 0, 1)
				for {
					link, err := client.Recv()
					if err == io.EOF {
						break
					}
					require.NoError(t, err)
					link.InstanceId = 0
					links = append(links, *link)
				}
				require.Equal(t, tt.want, grpcTest.SortResources(links))
			}
		})
	}
}
