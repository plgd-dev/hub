package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message"
)

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
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: deviceID,
							Href:     "/light/1",
						},
						Content: &commands.Content{
							CoapContentFormat: int32(message.AppOcfCbor),
							ContentType:       message.AppOcfCbor.String(),
							Data: test.EncodeToCbor(t, map[string]interface{}{
								"state": false,
								"power": uint64(0),
								"name":  "Light",
								"if":    []interface{}{"oic.if.rw", "oic.if.baseline"},
								"rt":    []interface{}{"core.light"},
							}),
						},
						Status: commands.Status_OK,
					},
				},
				{
					Types: []string{"core.light"},
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: deviceID,
							Href:     "/light/2",
						},
						Content: &commands.Content{
							CoapContentFormat: int32(message.AppOcfCbor),
							ContentType:       message.AppOcfCbor.String(),
							Data: test.EncodeToCbor(t, map[string]interface{}{
								"state": false,
								"power": uint64(0),
								"name":  "Light",
								"if":    []interface{}{"oic.if.rw", "oic.if.baseline"},
								"rt":    []interface{}{"core.light"},
							}),
						},
						Status: commands.Status_OK,
					},
				},
				{
					Types: []string{"oic.wk.con"},
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: deviceID,
							Href:     "/oc/con",
						},
						Content: &commands.Content{
							CoapContentFormat: int32(message.AppOcfCbor),
							ContentType:       message.AppOcfCbor.String(),
							Data: test.EncodeToCbor(t, map[string]interface{}{
								"n":  test.TestDeviceName,
								"if": []interface{}{"oic.if.rw", "oic.if.baseline"},
								"rt": []interface{}{"oic.wk.con"},
							}),
						},
						Status: commands.Status_OK,
					},
				},
				{
					Types: []string{"oic.wk.p"},
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: deviceID,
							Href:     "/oic/p",
						},
						Content: &commands.Content{
							CoapContentFormat: int32(message.AppOcfCbor),
							ContentType:       message.AppOcfCbor.String(),
							Data: test.EncodeToCbor(t, map[string]interface{}{
								"mnmn": "ocfcloud.com",
								//"pi":   "d9b71824-78f7-4f26-540b-d86eab696937",
								"if": []interface{}{"oic.if.r", "oic.if.baseline"},
								"rt": []interface{}{"oic.wk.p"},
							}),
						},
						Status: commands.Status_OK,
					},
				},
				{
					Types: []string{"oic.d.cloudDevice", "oic.wk.d"},
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: deviceID,
							Href:     "/oic/d",
						},
						Content: &commands.Content{
							CoapContentFormat: int32(message.AppOcfCbor),
							ContentType:       message.AppOcfCbor.String(),
							Data: test.EncodeToCbor(t, map[string]interface{}{
								"n":   test.TestDeviceName,
								"di":  deviceID,
								"dmv": "ocf.res.1.3.0",
								"icv": "ocf.2.0.5",
								// "piid": "1dcb14bd-5167-4122-6c2f-71741543fdc3",
								"if": []interface{}{"oic.if.r", "oic.if.baseline"},
								"rt": []interface{}{"oic.d.cloudDevice", "oic.wk.d"},
							}),
						},
						Status: commands.Status_OK,
					},
				},
			},
		},
		{
			name: "get oic.wk.d and oic.wk.p of " + deviceID,
			args: args{
				deviceID:   deviceID,
				typeFilter: []string{"oic.wk.d", "oic.wk.p"},
				accept:     uri.ApplicationProtoJsonContentType,
			},
			want: []*pb.Resource{
				{
					Types: []string{"oic.wk.p"},
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: deviceID,
							Href:     "/oic/p",
						},
						Content: &commands.Content{
							CoapContentFormat: int32(message.AppOcfCbor),
							ContentType:       message.AppOcfCbor.String(),
							Data: test.EncodeToCbor(t, map[string]interface{}{
								"mnmn": "ocfcloud.com",
								//"pi":   "d9b71824-78f7-4f26-540b-d86eab696937",
								"if": []interface{}{"oic.if.r", "oic.if.baseline"},
								"rt": []interface{}{"oic.wk.p"},
							}),
						},
						Status: commands.Status_OK,
					},
				},
				{
					Types: []string{"oic.d.cloudDevice", "oic.wk.d"},
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: deviceID,
							Href:     "/oic/d",
						},
						Content: &commands.Content{
							CoapContentFormat: int32(message.AppOcfCbor),
							ContentType:       message.AppOcfCbor.String(),
							Data: test.EncodeToCbor(t, map[string]interface{}{
								"n":   test.TestDeviceName,
								"di":  deviceID,
								"dmv": "ocf.res.1.3.0",
								"icv": "ocf.2.0.5",
								// "piid": "1dcb14bd-5167-4122-6c2f-71741543fdc3",
								"if": []interface{}{"oic.if.r", "oic.if.baseline"},
								"rt": []interface{}{"oic.d.cloudDevice", "oic.wk.d"},
							}),
						},
						Status: commands.Status_OK,
					},
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

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
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
