package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema/configuration"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/platform"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	test "github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
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

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resourceLinks := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, resourceLinks)
	defer shutdownDevSim()

	resourceLinks = append(resourceLinks, test.AddDeviceSwitchResources(ctx, t, deviceID, c, "1", "2", "3")...)
	time.Sleep(200 * time.Millisecond)

	type args struct {
		req *pb.GetResourceLinksRequest
	}
	tests := []struct {
		name string
		args args
		want []*events.ResourceLinksPublished
	}{
		{
			name: "valid",
			args: args{
				req: &pb.GetResourceLinksRequest{},
			},
			want: []*events.ResourceLinksPublished{
				{
					DeviceId:     deviceID,
					Resources:    test.ResourceLinksToResources(deviceID, resourceLinks),
					AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
				},
			},
		},
		{
			name: "invalid deviceFilter",
			args: args{
				req: &pb.GetResourceLinksRequest{
					DeviceIdFilter: []string{"unknown"},
				},
			},
			want: []*events.ResourceLinksPublished{},
		},
		{
			name: "valid deviceFilter",
			args: args{
				req: &pb.GetResourceLinksRequest{
					DeviceIdFilter: []string{deviceID},
				},
			},
			want: []*events.ResourceLinksPublished{
				{
					DeviceId:     deviceID,
					Resources:    test.ResourceLinksToResources(deviceID, resourceLinks),
					AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
				},
			},
		},
		{
			name: "invalid typefilter",
			args: args{
				req: &pb.GetResourceLinksRequest{
					TypeFilter: []string{"unknown"},
				},
			},
			want: []*events.ResourceLinksPublished{},
		},
		{
			name: "valid typefilter",
			args: args{
				req: &pb.GetResourceLinksRequest{
					TypeFilter: []string{platform.ResourceType, device.ResourceType, configuration.ResourceType},
				},
			},
			want: []*events.ResourceLinksPublished{
				{
					DeviceId:     deviceID,
					Resources:    test.ResourceLinksToResources(deviceID, resourceLinks[0:3]),
					AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetResourceLinks(ctx, tt.args.req)
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
