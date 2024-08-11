package service_test

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/plgd-dev/device/v2/pkg/codec/json"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func newBool(v bool) *bool {
	return &v
}

func TestRequestHandlerGetResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

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

	// get resource from device via HUB
	lightResourceData, err := c.GetResourceFromDevice(ctx, &pb.GetResourceFromDeviceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
	})
	require.NoError(t, err)

	type args struct {
		accept            string
		deviceID          string
		resourceHref      string
		resourceInterface string
		etag              []byte
		etags             [][]byte
		twin              *bool
	}
	tests := []struct {
		name     string
		args     args
		want     *events.ResourceRetrieved
		wantCode int
	}{
		{
			name: "json: get from resource twin",
			args: args{
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
			},
			want: &events.ResourceRetrieved{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     test.TestResourceLightInstanceHref("1"),
				},
				Status:        commands.Status_OK,
				Content:       &commands.Content{}, // content is encoded as json
				AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
				ResourceTypes: test.TestResourceLightInstanceResourceTypes,
			},
			wantCode: http.StatusOK,
		},
		{
			name: "jsonpb: get from resource twin",
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
			},
			want: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "",
				map[string]interface{}{
					"state": false,
					"power": uint64(0),
					"name":  "Light",
				},
			),
			wantCode: http.StatusOK,
		},
		{
			name: "jsonpb: get from resource twin with etag in header",
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				etag:         lightResourceData.GetData().GetEtag(),
			},
			wantCode: http.StatusNotModified,
		},
		{
			name: "jsonpb: get from resource twin with etag in queries",
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				etags:        [][]byte{[]byte(base64.StdEncoding.EncodeToString([]byte("abc"))), lightResourceData.GetData().GetEtag()},
			},
			wantCode: http.StatusNotModified,
		},
		{
			name: "jsonpb: get from resource twin with interface and etag",
			args: args{
				accept:            pkgHttp.ApplicationProtoJsonContentType,
				deviceID:          deviceID,
				resourceHref:      test.TestResourceLightInstanceHref("1"),
				resourceInterface: interfaces.OC_IF_BASELINE,
				etag:              lightResourceData.GetData().GetEtag(),
			},
			wantCode: http.StatusNotModified,
		},
		{
			name: "jsonpb: get from device with interface",
			args: args{
				accept:            pkgHttp.ApplicationProtoJsonContentType,
				deviceID:          deviceID,
				resourceHref:      test.TestResourceLightInstanceHref("1"),
				resourceInterface: interfaces.OC_IF_BASELINE,
			},
			want: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "",
				map[string]interface{}{
					"state": false,
					"power": uint64(0),
					"name":  "Light",
					"if":    []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
					"rt":    []interface{}{types.CORE_LIGHT},
				},
			),
			wantCode: http.StatusOK,
		},
		{
			name: "jsonpb: get from device with disabled twin",
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				twin:         newBool(false),
			},
			want: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "",
				map[string]interface{}{
					"state": false,
					"power": uint64(0),
					"name":  "Light",
				},
			),
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodGet, uri.AliasDeviceResource, nil).AuthToken(token).Accept(tt.args.accept)
			rb.DeviceId(tt.args.deviceID).ResourceHref(tt.args.resourceHref).ResourceInterface(tt.args.resourceInterface).ETag(tt.args.etag).ETags(tt.args.etags)
			if tt.args.twin != nil {
				rb.Twin(*tt.args.twin)
			}
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantCode, resp.StatusCode)
			values := make([]*events.ResourceRetrieved, 0, 1)
			for {
				var value pb.GetResourceFromDeviceResponse
				err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &value)
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				require.NotEmpty(t, value.GetData())
				values = append(values, value.GetData())
			}
			if tt.wantCode != http.StatusOK {
				require.Empty(t, values)
				return
			}
			require.Len(t, values, 1)
			pbTest.CmpResourceRetrieved(t, tt.want, values[0])
		})
	}
}

func TestRequestHandlerGetResourceWithOnlyContent(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

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

	type args struct {
		deviceID     string
		resourceHref string
		twin         *bool
	}
	tests := []struct {
		name     string
		args     args
		want     interface{}
		wantCode int
	}{
		{
			name: "json: get resource from twin",
			args: args{
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
			},
			want:     map[interface{}]interface{}{"name": "Light", "power": uint64(0x0), "state": false},
			wantCode: http.StatusOK,
		},
		{
			name: "json: get resource from device",
			args: args{
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				twin:         newBool(false),
			},
			want:     map[interface{}]interface{}{"name": "Light", "power": uint64(0x0), "state": false},
			wantCode: http.StatusOK,
		},
		{
			name: "json: not exists",
			args: args{
				deviceID:     deviceID,
				resourceHref: "/not-exists",
			},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodGet, uri.AliasDeviceResource, nil).AuthToken(token)
			rb.DeviceId(tt.args.deviceID).ResourceHref(tt.args.resourceHref)
			rb.OnlyContent(true)
			if tt.args.twin != nil {
				rb.Twin(*tt.args.twin)
			}
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantCode, resp.StatusCode)
			if tt.wantCode != http.StatusOK {
				return
			}
			var got interface{}
			err := json.ReadFrom(resp.Body, &got)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
