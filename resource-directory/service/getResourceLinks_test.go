package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerGetResourceLinks(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.GetResourceLinksRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*events.ResourceLinksPublished
	}{
		{
			name: "valid",
			args: args{
				req: &pb.GetResourceLinksRequest{},
			},
			wantErr: false,
			want: []*events.ResourceLinksPublished{
				{
					DeviceId:     deviceID,
					Resources:    test.ResourceLinksToResources(deviceID, test.GetAllBackendResourceLinks()),
					AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetResourceLinks(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			links := make([]*events.ResourceLinksPublished, 0, 1)
			for {
				link, err := client.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				require.NotEmpty(t, link.GetAuditContext())
				require.NotEmpty(t, link.GetEventMetadata())
				links = append(links, pbTest.CleanUpResourceLinksPublished(link, true))
			}
			test.CheckProtobufs(t, tt.want, links, test.RequireToCheckFunc(require.Equal))
		})
	}
}
