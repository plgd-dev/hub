package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/http-gateway/test"
	"github.com/plgd-dev/hub/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	test "github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthService "github.com/plgd-dev/hub/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/test/pb"
	"github.com/plgd-dev/hub/test/service"
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

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resourceLinks := test.GetAllBackendResourceLinks()
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, resourceLinks)
	defer shutdownDevSim()
	resourceLinks = append(resourceLinks, test.AddDeviceSwitchResources(ctx, t, deviceID, c, "1", "2", "3")...)
	time.Sleep(200 * time.Millisecond)

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	type args struct {
		deviceID string
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
					AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodGet, uri.AliasDeviceResourceLinks+"/", nil).DeviceId(deviceID).AuthToken(token).Build()
			resp := httpgwTest.HTTPDo(t, request)
			defer func() {
				_ = resp.Body.Close()
			}()

			var links []*events.ResourceLinksPublished
			for {
				var v events.ResourceLinksPublished
				err = Unmarshal(resp.StatusCode, resp.Body, &v)
				if err == io.EOF {
					break
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
