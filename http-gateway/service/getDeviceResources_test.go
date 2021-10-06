package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"testing"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/http-gateway/test"
	"github.com/plgd-dev/hub/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/pkg/ocf"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getResourceChanged(t *testing.T, deviceID, href string, data map[string]interface{}) *events.ResourceChanged {
	return &events.ResourceChanged{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Content: &commands.Content{
			CoapContentFormat: int32(message.AppOcfCbor),
			ContentType:       message.AppOcfCbor.String(),
			Data:              test.EncodeToCbor(t, data),
		},
		Status: commands.Status_OK,
	}
}

func getPlatformResourceChanged(t *testing.T, deviceID string) *events.ResourceChanged {
	return getResourceChanged(t, deviceID, test.OCFResourcePlatformHref,
		map[string]interface{}{
			"mnmn": "ocfcloud.com",
			//"pi":   "d9b71824-78f7-4f26-540b-d86eab696937",
			"if": []interface{}{ocf.OC_IF_R, ocf.OC_IF_BASELINE},
			"rt": []interface{}{ocf.OC_RT_P},
		},
	)
}

func getCloudDeviceResourceChanged(t *testing.T, deviceID string) *events.ResourceChanged {
	return getResourceChanged(t, deviceID, test.OCFResourceDeviceHref,
		map[string]interface{}{
			"n":   test.TestDeviceName,
			"di":  deviceID,
			"dmv": "ocf.res.1.3.0",
			"icv": "ocf.2.0.5",
			// "piid": "1dcb14bd-5167-4122-6c2f-71741543fdc3",
			"if": []interface{}{ocf.OC_IF_R, ocf.OC_IF_BASELINE},
			"rt": []interface{}{ocf.OC_RT_DEVICE_CLOUD, ocf.OC_RT_D},
		},
	)
}

func TestRequestHandler_GetDeviceResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
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
					Types: []string{"core.light"},
					Data: getResourceChanged(t, deviceID, test.TestResourceLightHref,
						map[string]interface{}{
							"state": false,
							"power": uint64(0),
							"name":  "Light",
							"if":    []interface{}{ocf.OC_IF_RW, ocf.OC_IF_BASELINE},
							"rt":    []interface{}{"core.light"},
						},
					),
				},
				{
					Types: []string{ocf.OC_RT_COL},
					Data: getResourceChanged(t, deviceID, test.TestResourceSwitchesHref,
						map[string]interface{}{
							"links":                     []interface{}{},
							"if":                        []interface{}{ocf.OC_IF_LL, ocf.OC_IF_CREATE, ocf.OC_IF_B, ocf.OC_IF_BASELINE},
							"rt":                        []interface{}{ocf.OC_RT_COL},
							"rts":                       []interface{}{ocf.OC_RT_RESOURCE_SWITCH},
							"rts-m":                     []interface{}{ocf.OC_RT_RESOURCE_SWITCH},
							"x.org.openconnectivity.bl": uint64(94),
						},
					),
				},
				{
					Types: []string{ocf.OC_RT_CON},
					Data: getResourceChanged(t, deviceID, test.OCFResourceConfigurationHref,
						map[string]interface{}{
							"n":  test.TestDeviceName,
							"if": []interface{}{ocf.OC_IF_RW, ocf.OC_IF_BASELINE},
							"rt": []interface{}{ocf.OC_RT_CON},
						},
					),
				},
				{
					Types: []string{ocf.OC_RT_P},
					Data:  getPlatformResourceChanged(t, deviceID),
				},
				{
					Types: []string{ocf.OC_RT_DEVICE_CLOUD, ocf.OC_RT_D},
					Data:  getCloudDeviceResourceChanged(t, deviceID),
				},
			},
		},
		{
			name: "get oic.wk.d and oic.wk.p of " + deviceID,
			args: args{
				deviceID:   deviceID,
				typeFilter: []string{ocf.OC_RT_D, ocf.OC_RT_P},
				accept:     uri.ApplicationProtoJsonContentType,
			},
			want: []*pb.Resource{
				{
					Types: []string{ocf.OC_RT_P},
					Data:  getPlatformResourceChanged(t, deviceID),
				},
				{
					Types: []string{ocf.OC_RT_DEVICE_CLOUD, ocf.OC_RT_D},
					Data:  getCloudDeviceResourceChanged(t, deviceID),
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

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodGet, uri.AliasDeviceResources, nil).DeviceId(tt.args.deviceID).Accept(tt.args.accept).AddTypeFilter(tt.args.typeFilter).AuthToken(token).Build()
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

			values := make([]*pb.Resource, 0, 1)
			for {
				var value pb.Resource
				err = Unmarshal(resp.StatusCode, resp.Body, &value)
				if err == io.EOF {
					break
				}
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				value.Data.AuditContext = nil
				value.Data.EventMetadata = nil
				values = append(values, &value)
			}
			cmpResourceValues(t, tt.want, values)
		})
	}
}
