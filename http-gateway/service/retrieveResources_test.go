package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerGetResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resourceLinks := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.COAPS_TCP_SCHEME+config.COAP_GW_HOST, resourceLinks)
	defer shutdownDevSim()
	const switchID = "1"
	resourceLinks = append(resourceLinks, test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)...)
	time.Sleep(200 * time.Millisecond)

	type args struct {
		accept           string
		deviceIdFilter   []string
		resourceIdFilter []string
		typeFilter       []string
	}
	tests := []struct {
		name  string
		args  args
		cmpFn func(*testing.T, []*pb.Resource, []*pb.Resource)
		want  []*pb.Resource
	}{
		{
			name: "invalid deviceIdFilter",
			args: args{
				accept:         uri.ApplicationProtoJsonContentType,
				deviceIdFilter: []string{"unknown"},
			},
		},
		{
			name: "invalid resourceIdFilter",
			args: args{
				accept:           uri.ApplicationProtoJsonContentType,
				resourceIdFilter: []string{"unknown"},
			},
		},
		{
			name: "invalid typeFilter",
			args: args{
				accept:     uri.ApplicationProtoJsonContentType,
				typeFilter: []string{"unknown"},
			},
		},
		{
			name: "valid deviceIdFilter",
			args: args{
				accept:         uri.ApplicationProtoJsonContentType,
				deviceIdFilter: []string{deviceID},
			},
			cmpFn: pbTest.CmpResourceValuesBasic,
			want:  test.ResourceLinksToResources2(deviceID, resourceLinks),
		},
		{
			name: "valid resourceIdFilter",
			args: args{
				accept: uri.ApplicationProtoJsonContentType,
				resourceIdFilter: []string{
					commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")).ToString(),
				},
			},
			want: []*pb.Resource{
				{
					Types: []string{types.CORE_LIGHT},
					Data: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceLightInstanceHref("1"), "",
						map[string]interface{}{
							"state": false,
							"power": uint64(0),
							"name":  "Light",
						},
					),
				},
			},
		},
		{
			name: "valid typeFilter",
			args: args{
				accept:     uri.ApplicationProtoJsonContentType,
				typeFilter: []string{types.BINARY_SWITCH},
			},
			want: []*pb.Resource{
				{
					Types: []string{types.BINARY_SWITCH},
					Data: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesInstanceHref(switchID), "",
						map[string]interface{}{
							"value": false,
						},
					),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodGet, uri.Resources, nil).AuthToken(token).Accept(tt.args.accept)
			rb.AddDeviceIdFilter(tt.args.deviceIdFilter).AddResourceIdFilter(tt.args.resourceIdFilter).AddTypeFilter(tt.args.typeFilter)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()

			values := make([]*pb.Resource, 0, 1)
			for {
				var value pb.Resource
				err = httpgwTest.Unmarshal(resp.StatusCode, resp.Body, &value)
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				values = append(values, &value)
			}
			if tt.cmpFn != nil {
				tt.cmpFn(t, tt.want, values)
				return
			}
			pbTest.CmpResourceValues(t, tt.want, values)
		})
	}
}
