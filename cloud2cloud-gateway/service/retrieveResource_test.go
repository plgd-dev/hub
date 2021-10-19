package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/cloud2cloud-gateway/uri"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	testHttp "github.com/plgd-dev/hub/test/http"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandler_RetrieveResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		uri    string
		accept string
	}
	tests := []struct {
		name            string
		args            args
		wantContentType string
		wantCode        int
		want            interface{}
	}{
		{
			name: "JSON: " + uri.Devices + "/" + deviceID + test.TestResourceLightInstanceHref("1"),
			args: args{
				uri:    "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + test.TestResourceLightInstanceHref("1"),
				accept: message.AppJSON.String(),
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want: map[interface{}]interface{}{
				"if":    []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
				"name":  "Light",
				"power": uint64(0),
				"state": false,
				"rt":    []interface{}{types.CORE_LIGHT},
			},
		},
		{
			name: "CBOR: " + uri.Devices + "/" + deviceID + test.TestResourceLightInstanceHref("1"),
			args: args{
				uri:    "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + test.TestResourceLightInstanceHref("1"),
				accept: message.AppOcfCbor.String(),
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppOcfCbor.String(),
			want: map[interface{}]interface{}{
				"if":    []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
				"name":  "Light",
				"power": uint64(0),
				"state": false,
				"rt":    []interface{}{types.CORE_LIGHT},
			},
		},
		{
			name: "notFound",
			args: args{
				uri:    "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + "/notFound",
				accept: message.AppJSON.String(),
			},
			wantCode:        http.StatusNotFound,
			wantContentType: "text/plain",
			want:            "cannot retrieve resource: cannot retrieve resource(deviceID: " + deviceID + ", Href: /notFound): rpc error: code = NotFound desc = cannot retrieve resources values: not found",
		},
		{
			name: "invalidAccept",
			args: args{
				uri:    "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + test.TestResourceLightInstanceHref("1"),
				accept: "application/invalid",
			},
			wantCode:        http.StatusBadRequest,
			wantContentType: "text/plain",
			want:            "cannot retrieve resource: invalid accept header([application/invalid])",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testHttp.NewHTTPRequest(http.MethodGet, tt.args.uri, nil).Accept(tt.args.accept).Build(ctx, t)
			resp := testHttp.DoHTTPRequest(t, req)
			assert.Equal(t, tt.wantCode, resp.StatusCode)
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantContentType, resp.Header.Get("Content-Type"))
			if tt.want != nil {
				got := testHttp.ReadHTTPResponse(t, resp.Body, tt.wantContentType)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
