package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	c2cTest "github.com/plgd-dev/hub/v2/cloud2cloud-gateway/test"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/uri"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerRetrieveResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(testCfg.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.ACTIVE_COAP_SCHEME+testCfg.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	const textPlain = "text/plain"
	type args struct {
		accept       string
		token        string
		resourceHref string
	}
	tests := []struct {
		name            string
		args            args
		wantContentType string
		wantCode        int
		want            interface{}
	}{
		{
			name: "missing token",
			args: args{
				resourceHref: test.TestResourceSwitchesHref,
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: token contains an invalid number of segments",
		},
		{
			name: "expired token",
			args: args{
				token:        oauthTest.GetAccessToken(t, testCfg.OAUTH_SERVER_HOST, oauthTest.ClientTestExpired),
				resourceHref: test.TestResourceSwitchesHref,
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: token is expired",
		},
		{
			name: "notFound",
			args: args{
				accept:       message.AppJSON.String(),
				token:        token,
				resourceHref: "/notFound",
			},
			wantCode:        http.StatusNotFound,
			wantContentType: textPlain,
			want:            "cannot retrieve resource: cannot retrieve resource(deviceID: " + deviceID + ", Href: /notFound): rpc error: code = NotFound desc = cannot retrieve resources values: not found",
		},
		{
			name: "invalidAccept",
			args: args{
				accept:       "application/invalid",
				token:        token,
				resourceHref: test.TestResourceLightInstanceHref("1"),
			},
			wantCode:        http.StatusBadRequest,
			wantContentType: textPlain,
			want:            "cannot retrieve resource: invalid accept header([application/invalid])",
		},
		{
			name: "JSON: " + uri.Devices + "/" + deviceID + test.TestResourceLightInstanceHref("1"),
			args: args{
				accept:       message.AppJSON.String(),
				token:        token,
				resourceHref: test.TestResourceLightInstanceHref("1"),
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want: map[interface{}]interface{}{
				"name":  "Light",
				"power": uint64(0),
				"state": false,
			},
		},
		{
			name: "CBOR: " + uri.Devices + "/" + deviceID + test.TestResourceLightInstanceHref("1"),
			args: args{
				accept:       message.AppOcfCbor.String(),
				token:        token,
				resourceHref: test.TestResourceLightInstanceHref("1"),
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppOcfCbor.String(),
			want: map[interface{}]interface{}{
				"name":  "Light",
				"power": uint64(0),
				"state": false,
			},
		},
		{
			name: "JSON: " + uri.Devices + "/" + deviceID + test.TestResourceSwitchesHref,
			args: args{
				accept:       message.AppJSON.String(),
				token:        token,
				resourceHref: test.TestResourceSwitchesHref,
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want:            []interface{}{},
		},
		{
			name: "CBOR: " + uri.Devices + "/" + deviceID + test.TestResourceSwitchesHref,
			args: args{
				accept:       message.AppOcfCbor.String(),
				token:        token,
				resourceHref: test.TestResourceSwitchesHref,
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppOcfCbor.String(),
			want:            []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := testHttp.NewHTTPRequest(http.MethodGet, c2cTest.C2CURI(uri.ResourceValues), nil).Accept(tt.args.accept).AuthToken(tt.args.token)
			rb.DeviceId(deviceID).ResourceHref(tt.args.resourceHref)
			resp := testHttp.DoHTTPRequest(t, rb.Build(ctx, t))
			assert.Equal(t, tt.wantCode, resp.StatusCode)
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantContentType, resp.Header.Get("Content-Type"))
			got := testHttp.ReadHTTPResponse(t, resp.Body, tt.wantContentType)
			if tt.wantContentType == textPlain {
				require.Contains(t, got, tt.want)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}
