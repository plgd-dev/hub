package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	exCodes "github.com/plgd-dev/hub/grpc-gateway/pb/codes"
	httpgwTest "github.com/plgd-dev/hub/http-gateway/test"
	"github.com/plgd-dev/hub/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/test/pb"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func TestRequestHandlerDeleteResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		accept string
		href   string
		ttl    time.Duration
	}
	tests := []struct {
		name         string
		args         args
		want         *events.ResourceDeleted
		wantErr      bool
		wantErrCode  exCodes.Code
		wantHTTPCode int
	}{
		{
			name: "/light/1 - MethodNotAllowed",
			args: args{
				accept: uri.ApplicationProtoJsonContentType,
				href:   test.TestResourceLightInstanceHref("1"),
			},
			wantErr:      true,
			wantErrCode:  exCodes.MethodNotAllowed,
			wantHTTPCode: http.StatusMethodNotAllowed,
		},
		{
			name: "invalid Href",
			args: args{
				accept: uri.ApplicationProtoJsonContentType,
				href:   "/unknown",
			},
			wantErr:      true,
			wantErrCode:  exCodes.Code(codes.NotFound),
			wantHTTPCode: http.StatusNotFound,
		},
		{
			name: "/oic/d - PermissionDenied",
			args: args{
				accept: uri.ApplicationProtoJsonContentType,
				href:   device.ResourceURI,
			},
			wantErr:      true,
			wantErrCode:  exCodes.Code(codes.PermissionDenied),
			wantHTTPCode: http.StatusForbidden,
		},
		{
			name: "invalid timeToLive",
			args: args{
				accept: uri.ApplicationProtoJsonContentType,
				href:   test.TestResourceLightInstanceHref("1"),
				ttl:    99 * time.Millisecond,
			},
			wantErr:      true,
			wantErrCode:  exCodes.Code(codes.InvalidArgument),
			wantHTTPCode: http.StatusBadRequest,
		},
		{
			name: "not found - delete /switches/-1",
			args: args{
				accept: uri.ApplicationProtoJsonContentType,
				href:   test.TestResourceSwitchesInstanceHref("-1"),
			},
			wantErr:      true,
			wantErrCode:  exCodes.Code(codes.NotFound),
			wantHTTPCode: http.StatusNotFound,
		},
		{
			name: "delete /switches/1",
			args: args{
				accept: uri.ApplicationProtoJsonContentType,
				href:   test.TestResourceSwitchesInstanceHref("1"),
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

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, "1", "2", "3")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodDelete, uri.DeviceResourceLink, nil).AuthToken(token).Accept(tt.args.accept)
			rb.DeviceId(deviceID).ResourceHref(tt.args.href).AddTimeToLive(tt.args.ttl)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()
			assert.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got pb.DeleteResourceResponse
			err = Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrCode.String(), exCodes.Code(status.Convert(err).Code()).String())
				return
			}
			require.NoError(t, err)
			want := pbTest.MakeResourceDeleted(t, deviceID, tt.args.href)
			pbTest.CmpResourceDeleted(t, want, got.GetData())
		})
	}
}
