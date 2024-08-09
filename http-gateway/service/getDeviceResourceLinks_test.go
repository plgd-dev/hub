package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/collection"
	"github.com/plgd-dev/device/v2/test/resource/types"
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

func TestRequestHandlerGetDeviceResourceLinks(t *testing.T) {
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
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resourceLinks := test.GetAllBackendResourceLinks()
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, resourceLinks)
	defer shutdownDevSim()
	resourceLinks = append(resourceLinks, test.AddDeviceSwitchResources(ctx, t, deviceID, c, "1", "2", "3")...)
	time.Sleep(200 * time.Millisecond)

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	type args struct {
		deviceID   string
		typeFilter []string
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
				deviceID: deviceID,
			},
			wantErr: false,
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
			wantErr: true,
		},
		{
			name: "valid typefilter",
			args: args{
				typeFilter: []string{collection.ResourceType, types.BINARY_SWITCH},
			},
			want: []*events.ResourceLinksPublished{
				{
					DeviceId: deviceID,
					Resources: test.ResourceLinksToResources(deviceID, test.FilterResourceLink(func(rl schema.ResourceLink) bool {
						return slices.Contains(rl.ResourceTypes, collection.ResourceType) ||
							slices.Contains(rl.ResourceTypes, types.BINARY_SWITCH)
					}, resourceLinks)),
					AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodGet, uri.AliasDeviceResourceLinks+"/", nil).DeviceId(deviceID).AuthToken(token).AddTypeFilter(tt.args.typeFilter).Build()
			resp := httpgwTest.HTTPDo(t, request)
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
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.NotEmpty(t, v.GetAuditContext())
				require.NotEmpty(t, v.GetEventMetadata())
				links = append(links, pbTest.CleanUpResourceLinksPublished(&v, true))
			}
			test.CheckProtobufs(t, tt.want, links, test.RequireToCheckFunc(require.Equal))
		})
	}
}
