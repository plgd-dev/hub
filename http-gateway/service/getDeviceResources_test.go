package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/collection"
	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func makePlatformResourceChanged(t *testing.T, deviceID string) *events.ResourceChanged {
	return pbTest.MakeResourceChanged(t, deviceID, platform.ResourceURI, "",
		map[string]interface{}{
			"mnmn": "ocfcloud.com",
		},
	)
}

func makeCloudDeviceResourceChanged(t *testing.T, deviceID string) *events.ResourceChanged {
	return pbTest.MakeResourceChanged(t, deviceID, device.ResourceURI, "",
		map[string]interface{}{
			"n":   test.TestDeviceName,
			"di":  deviceID,
			"dmv": "ocf.res.1.3.0",
			"icv": "ocf.2.0.5",
		},
	)
}

func TestRequestHandlerGetDeviceResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	const switchID = "1"
	type args struct {
		deviceID   string
		typeFilter []string
		accept     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.Resource
	}{
		{
			name: "get resource of " + deviceID,
			args: args{
				deviceID: deviceID,
				accept:   uri.ApplicationProtoJsonContentType,
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
				{
					Types: []string{collection.ResourceType},
					Data: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesHref, "",
						[]interface{}{
							map[string]interface{}{
								"href": test.TestResourceSwitchesInstanceHref(switchID),
								"if":   []string{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
								"p": map[string]interface{}{
									"bm": uint64(schema.Discoverable | schema.Observable),
								},
								"rel": []interface{}{"hosts"},
								"rt":  []interface{}{types.BINARY_SWITCH},
							},
						},
					),
				},
				{
					Types: []string{types.BINARY_SWITCH},
					Data: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesInstanceHref(switchID), "",
						map[string]interface{}{
							"value": false,
						}),
				},
				{
					Types: []string{configuration.ResourceType},
					Data: pbTest.MakeResourceChanged(t, deviceID, configuration.ResourceURI, "",
						map[string]interface{}{
							"n": test.TestDeviceName,
						},
					),
				},
				{
					Types: []string{platform.ResourceType},
					Data:  makePlatformResourceChanged(t, deviceID),
				},
				{
					Types: []string{types.DEVICE_CLOUD, device.ResourceType},
					Data:  makeCloudDeviceResourceChanged(t, deviceID),
				},
			},
		},
		{
			name: "get oic.wk.d and oic.wk.p of " + deviceID,
			args: args{
				deviceID:   deviceID,
				typeFilter: []string{device.ResourceType, platform.ResourceType},
				accept:     uri.ApplicationProtoJsonContentType,
			},
			want: []*pb.Resource{
				{
					Types: []string{platform.ResourceType},
					Data:  makePlatformResourceChanged(t, deviceID),
				},
				{
					Types: []string{types.DEVICE_CLOUD, device.ResourceType},
					Data:  makeCloudDeviceResourceChanged(t, deviceID),
				},
			},
		},
		{
			name: "not found",
			args: args{
				deviceID: "notFound",
				accept:   uri.ApplicationProtoJsonContentType,
			},
			wantErr: true,
		},
	}

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
	defer func() {
		_ = conn.Close()
	}()
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)
	time.Sleep(time.Millisecond * 200)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodGet, uri.AliasDeviceResources, nil).Accept(tt.args.accept).AuthToken(token)
			rb.DeviceId(tt.args.deviceID).AddTypeFilter(tt.args.typeFilter)
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
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				values = append(values, &value)
			}
			pbTest.CmpResourceValues(t, tt.want, values)
		})
	}
}
