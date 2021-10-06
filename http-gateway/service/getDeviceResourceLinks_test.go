package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/cloud/v2/http-gateway/test"
	"github.com/plgd-dev/cloud/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/events"
	test "github.com/plgd-dev/cloud/v2/test"
	"github.com/plgd-dev/cloud/v2/test/config"
	oauthTest "github.com/plgd-dev/cloud/v2/test/oauth-server/test"
)

func TestRequestHandler_GetDeviceResourceLinks(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
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
					DeviceId:  deviceID,
					Resources: test.ResourceLinksToResources(deviceID, test.GetAllBackendResourceLinks()),
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodGet, uri.AliasDeviceResourceLinks+"/", nil).DeviceId(deviceID).AuthToken(token).Build()
			trans := http.DefaultTransport.(*http.Transport).Clone()
			trans.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
			c := http.Client{
				Transport: trans,
			}
			resp, err := c.Do(request)
			require.NoError(t, err)
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
				links = append(links, test.CleanUpResourceLinksPublished(&v))
			}
			test.CheckProtobufs(t, tt.want, links, test.RequireToCheckFunc(require.Equal))
		})
	}
}
