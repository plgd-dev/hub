package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/codec/json"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerUpdateResourcesValues(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	switchID := "1"
	type args struct {
		accept            string
		contentType       string
		deviceID          string
		resourceHref      string
		resourceInterface string
		onlyContent       bool
		resourceData      interface{}
		ttl               time.Duration
	}
	tests := []struct {
		name         string
		args         args
		want         *events.ResourceUpdated
		wantErr      bool
		wantHTTPCode int
	}{
		{
			name: "invalid href",
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				contentType:  message.AppJSON.String(),
				deviceID:     deviceID,
				resourceHref: "/unknown",
				onlyContent:  true,
			},
			wantErr:      true,
			wantHTTPCode: http.StatusNotFound,
		},
		{
			name: "invalid timeToLive",
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				contentType:  message.AppJSON.String(),
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				ttl:          99 * time.Millisecond,
			},
			wantErr:      true,
			wantHTTPCode: http.StatusBadRequest,
		},
		{
			name: "invalid update RO-resource",
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				contentType:  message.AppJSON.String(),
				deviceID:     deviceID,
				resourceHref: device.ResourceURI,
			},
			wantErr:      true,
			wantHTTPCode: http.StatusForbidden,
		},
		{
			name: "invalid update collection /switches",
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				contentType:  message.AppJSON.String(),
				deviceID:     deviceID,
				resourceHref: test.TestResourceSwitchesHref,
			},
			wantErr:      true,
			wantHTTPCode: http.StatusForbidden,
		},
		{
			name: "valid - " + pkgHttp.ApplicationProtoJsonContentType,
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				contentType:  pkgHttp.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				resourceData: map[string]interface{}{
					"power": 1,
				},
			},
			want:         pbTest.MakeResourceUpdated(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "", nil),
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "valid - " + message.AppJSON.String(),
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				contentType:  message.AppJSON.String(),
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				resourceData: map[string]interface{}{
					"power": 102,
				},
			},
			want:         pbTest.MakeResourceUpdated(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "", nil),
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "valid with interface",
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				contentType:  message.AppJSON.String(),
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				resourceData: map[string]interface{}{
					"power": 2,
				},
				resourceInterface: interfaces.OC_IF_BASELINE,
			},
			want:         pbTest.MakeResourceUpdated(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "", nil),
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "revert update",
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				contentType:  message.AppJSON.String(),
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				resourceData: map[string]interface{}{
					"power": 0,
				},
				resourceInterface: interfaces.OC_IF_BASELINE,
			},
			want:         pbTest.MakeResourceUpdated(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "", nil),
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "update /switches/1",
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				contentType:  message.AppJSON.String(),
				deviceID:     deviceID,
				resourceHref: test.TestResourceSwitchesInstanceHref(switchID),
				resourceData: map[string]interface{}{
					"value": true,
				},
			},
			want: &events.ResourceUpdated{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     test.TestResourceSwitchesInstanceHref(switchID),
				},
				Status: commands.Status_OK,
				Content: &commands.Content{
					CoapContentFormat: int32(message.AppOcfCbor),
					ContentType:       message.AppOcfCbor.String(),
					Data: test.EncodeToCbor(t, map[string]interface{}{
						"value": true,
					}),
				},
				AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
				ResourceTypes: test.TestResourceSwitchesInstanceResourceTypes,
			},
			wantHTTPCode: http.StatusOK,
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

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)
	time.Sleep(200 * time.Millisecond)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := httpTest.GetContentData(&pb.Content{
				Data:        test.EncodeToCbor(t, tt.args.resourceData),
				ContentType: message.AppOcfCbor.String(),
			}, tt.args.contentType)
			require.NoError(t, err)
			rb := httpgwTest.NewRequest(http.MethodPut, uri.AliasDeviceResource, bytes.NewReader(data)).AuthToken(token).Accept(tt.args.accept)
			rb.DeviceId(tt.args.deviceID).ResourceHref(tt.args.resourceHref).ResourceInterface(tt.args.resourceInterface).ContentType(tt.args.contentType)
			rb.AddTimeToLive(tt.args.ttl).OnlyContent(tt.args.onlyContent)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()
			assert.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got pb.UpdateResourceResponse
			err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			pbTest.CmpResourceUpdated(t, tt.want, got.GetData())
		})
	}
}

func TestRequestHandlerUpdateResourcesValuesWithOnlyContent(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	switchID := "1"
	type args struct {
		accept       string
		contentType  string
		deviceID     string
		resourceHref string
		resourceData interface{}
	}
	tests := []struct {
		name         string
		args         args
		want         interface{}
		wantErr      bool
		wantHTTPCode int
	}{
		{
			name: "valid - accept " + pkgHttp.ApplicationProtoJsonContentType,
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				contentType:  message.AppJSON.String(),
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				resourceData: map[string]interface{}{
					"power": 102,
				},
			},
			want:         nil,
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "revert-update - accept empty",
			args: args{
				contentType:  message.AppJSON.String(),
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				resourceData: map[string]interface{}{
					"power": 0,
				},
			},
			want:         nil,
			wantHTTPCode: http.StatusOK,
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

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)
	time.Sleep(200 * time.Millisecond)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := httpTest.GetContentData(&pb.Content{
				Data:        test.EncodeToCbor(t, tt.args.resourceData),
				ContentType: message.AppOcfCbor.String(),
			}, tt.args.contentType)
			require.NoError(t, err)
			rb := httpgwTest.NewRequest(http.MethodPut, uri.AliasDeviceResource, bytes.NewReader(data)).AuthToken(token).Accept(tt.args.accept)
			rb.DeviceId(tt.args.deviceID).ResourceHref(tt.args.resourceHref).ContentType(tt.args.contentType)
			rb.OnlyContent(true)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()
			assert.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got interface{}
			err = json.ReadFrom(resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
