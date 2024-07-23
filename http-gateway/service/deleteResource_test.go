package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	exCodes "github.com/plgd-dev/hub/v2/grpc-gateway/pb/codes"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
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
				accept: pkgHttp.ApplicationProtoJsonContentType,
				href:   test.TestResourceLightInstanceHref("1"),
			},
			wantErr:      true,
			wantErrCode:  exCodes.MethodNotAllowed,
			wantHTTPCode: http.StatusMethodNotAllowed,
		},
		{
			name: "invalid Href",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				href:   "/unknown",
			},
			wantErr:      true,
			wantErrCode:  exCodes.Code(codes.NotFound),
			wantHTTPCode: http.StatusNotFound,
		},
		{
			name: "/oic/d - PermissionDenied",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				href:   device.ResourceURI,
			},
			wantErr:      true,
			wantErrCode:  exCodes.Code(codes.PermissionDenied),
			wantHTTPCode: http.StatusForbidden,
		},
		{
			name: "invalid timeToLive",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
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
				accept: pkgHttp.ApplicationProtoJsonContentType,
				href:   test.TestResourceSwitchesInstanceHref("-1"),
			},
			wantErr:      true,
			wantErrCode:  exCodes.Code(codes.NotFound),
			wantHTTPCode: http.StatusNotFound,
		},
		{
			name: "delete /switches/1",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
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
			err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrCode.String(), exCodes.Code(status.Convert(err).Code()).String())
				return
			}
			require.NoError(t, err)
			want := pbTest.MakeResourceDeleted(deviceID, tt.args.href, test.TestResourceSwitchesInstanceResourceTypes, "")
			pbTest.CmpResourceDeleted(t, want, got.GetData())
		})
	}
}

func TestRequestHandlerBatchDeleteResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	switchIDs := []string{"1", "2", "3", "4", "5", "6", "7", "8"}

	type args struct {
		accept string
		href   string
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
			name: "/oic/res - Delete not supported",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				href:   resources.ResourceURI,
			},
			wantErr:      true,
			wantErrCode:  exCodes.Code(codes.NotFound),
			wantHTTPCode: http.StatusNotFound,
		},
		{
			name: "/switches/1 - Batch delete not supported",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				href:   test.TestResourceSwitchesInstanceHref("1"),
			},
			wantErr:      true,
			wantErrCode:  exCodes.Code(codes.PermissionDenied),
			wantHTTPCode: http.StatusForbidden,
		},
		{
			name: "/switches",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
				href:   test.TestResourceSwitchesHref,
			},
			want: func() *events.ResourceDeleted {
				rdel := pbTest.MakeResourceDeleted(deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, "")
				links := test.CollectionLinkRepresentations{}
				for _, switchID := range switchIDs {
					links = append(links, test.CollectionLinkRepresentation{
						Href:           test.TestResourceSwitchesInstanceHref(switchID),
						Representation: map[interface{}]interface{}{},
					})
				}
				rdel.Content = &commands.Content{
					CoapContentFormat: int32(message.AppOcfCbor),
					ContentType:       message.AppOcfCbor.String(),
					Data:              test.EncodeToCbor(t, links),
				}
				return rdel
			}(),
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
	_, shutdown := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdown()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchIDs...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodDelete, uri.DeviceResourceLink, nil).AuthToken(token).Accept(tt.args.accept)
			rb.DeviceId(deviceID).ResourceHref(tt.args.href).ResourceInterface(interfaces.OC_IF_B)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()
			assert.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got pb.DeleteResourceResponse
			err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrCode.String(), exCodes.Code(status.Convert(err).Code()).String())
				return
			}
			require.NoError(t, err)
			pbTest.CmpResourceDeleted(t, tt.want, got.GetData())
		})
	}
}
