package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/http-gateway/test"
	"github.com/plgd-dev/hub/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthService "github.com/plgd-dev/hub/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/test/pb"
	"github.com/plgd-dev/hub/test/service"
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
				accept:       uri.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: "/unknown",
			},
			wantErr:      true,
			wantHTTPCode: http.StatusNotFound,
		},
		{
			name: "invalid timeToLive",
			args: args{
				accept:       uri.ApplicationProtoJsonContentType,
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
				accept:       uri.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: device.ResourceURI,
			},
			wantErr:      true,
			wantHTTPCode: http.StatusForbidden,
		},
		{
			name: "invalid update collection /switches",
			args: args{
				accept:       uri.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: test.TestResourceSwitchesHref,
			},
			wantErr:      true,
			wantHTTPCode: http.StatusForbidden,
		},
		{
			name: "valid - " + uri.ApplicationProtoJsonContentType,
			args: args{
				accept:       uri.ApplicationProtoJsonContentType,
				contentType:  uri.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				resourceData: map[string]interface{}{
					"power": 1,
				},
			},
			want:         pbTest.MakeResourceUpdated(deviceID, test.TestResourceLightInstanceHref("1"), ""),
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "valid - " + message.AppJSON.String(),
			args: args{
				accept:       uri.ApplicationProtoJsonContentType,
				contentType:  message.AppJSON.String(),
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				resourceData: map[string]interface{}{
					"power": 102,
				},
			},
			want:         pbTest.MakeResourceUpdated(deviceID, test.TestResourceLightInstanceHref("1"), ""),
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "valid with interface",
			args: args{
				accept:       uri.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				resourceData: map[string]interface{}{
					"power": 2,
				},
				resourceInterface: interfaces.OC_IF_BASELINE,
			},
			want:         pbTest.MakeResourceUpdated(deviceID, test.TestResourceLightInstanceHref("1"), ""),
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "revert update",
			args: args{
				accept:       uri.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				resourceData: map[string]interface{}{
					"power": 0,
				},
				resourceInterface: interfaces.OC_IF_BASELINE,
			},
			want:         pbTest.MakeResourceUpdated(deviceID, test.TestResourceLightInstanceHref("1"), ""),
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "update /switches/1",
			args: args{
				accept:       uri.ApplicationProtoJsonContentType,
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
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
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

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)
	time.Sleep(200 * time.Millisecond)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := httpgwTest.GetContentData(&pb.Content{
				Data:        test.EncodeToCbor(t, tt.args.resourceData),
				ContentType: message.AppOcfCbor.String(),
			}, tt.args.contentType)
			require.NoError(t, err)
			rb := httpgwTest.NewRequest(http.MethodPut, uri.AliasDeviceResource, bytes.NewReader([]byte(data))).AuthToken(token).Accept(tt.args.accept)
			rb.DeviceId(tt.args.deviceID).ResourceHref(tt.args.resourceHref).ResourceInterface(tt.args.resourceInterface).ContentType(tt.args.contentType)
			rb.AddTimeToLive(tt.args.ttl)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()
			assert.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got pb.UpdateResourceResponse
			err = Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			pbTest.CmpResourceUpdated(t, tt.want, got.GetData())
		})
	}
}
