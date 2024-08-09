package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	test "github.com/plgd-dev/hub/v2/test"
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	defer func() {
		_ = conn.Close()
	}()
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	resourceLinks := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, resourceLinks)
	defer shutdownDevSim()
	resourceLinks = append(resourceLinks, test.AddDeviceSwitchResources(ctx, t, deviceID, c, "1", "2", "3")...)
	time.Sleep(200 * time.Millisecond)

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	type args struct {
		typeFilter []string
	}

	tests := []struct {
		name string
		args args
		want []*events.ResourceLinksPublished
	}{
		{
			name: "valid",
			args: args{},
			want: []*events.ResourceLinksPublished{
				{
					DeviceId:     deviceID,
					Resources:    test.ResourceLinksToResources(deviceID, resourceLinks),
					AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
				},
			},
		},
		{
			name: "invalid typefilter",
			args: args{
				typeFilter: []string{"unknown"},
			},
			want: nil,
		},
		{
			name: "valid typefilter",
			args: args{
				typeFilter: []string{platform.ResourceType, device.ResourceType, configuration.ResourceType},
			},
			want: []*events.ResourceLinksPublished{
				{
					DeviceId:     deviceID,
					Resources:    test.ResourceLinksToResources(deviceID, resourceLinks[0:3]),
					AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodGet, uri.ResourceLinks, nil).AuthToken(token).AddTypeFilter(tt.args.typeFilter)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()

			var links []*events.ResourceLinksPublished
			for {
				var v events.ResourceLinksPublished
				err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &v)
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				links = append(links, pbTest.CleanUpResourceLinksPublished(&v, true))
			}
			test.CheckProtobufs(t, tt.want, links, test.RequireToCheckFunc(require.Equal))
		})
	}
}
