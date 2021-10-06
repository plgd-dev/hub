package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	exCodes "github.com/plgd-dev/hub/grpc-gateway/pb/codes"
	httpgwTest "github.com/plgd-dev/hub/http-gateway/test"
	"github.com/plgd-dev/hub/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func TestRequestHandler_DeleteResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		deviceID     string
		resourceHref string
		accept       string
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
				deviceID:     deviceID,
				resourceHref: "/light/1",
				accept:       uri.ApplicationProtoJsonContentType,
			},
			wantErr:      true,
			wantErrCode:  exCodes.MethodNotAllowed,
			wantHTTPCode: http.StatusMethodNotAllowed,
		},
		{
			name: "invalid Href",
			args: args{
				deviceID:     deviceID,
				resourceHref: "/unknown",
				accept:       uri.ApplicationProtoJsonContentType,
			},
			wantErr:      true,
			wantErrCode:  exCodes.Code(codes.NotFound),
			wantHTTPCode: http.StatusNotFound,
		},
		{
			name: "/oic/d - PermissionDenied",
			args: args{
				deviceID:     deviceID,
				resourceHref: "/oic/d",
				accept:       uri.ApplicationProtoJsonContentType,
			},
			wantErr:      true,
			wantErrCode:  exCodes.Code(codes.PermissionDenied),
			wantHTTPCode: http.StatusForbidden,
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
			request := httpgwTest.NewRequest(http.MethodDelete, uri.DeviceResourceLink, nil).DeviceId(tt.args.deviceID).ResourceHref(tt.args.resourceHref).AuthToken(token).Accept(tt.args.accept).Build()
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

			assert.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got events.ResourceDeleted
			err = Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrCode.String(), exCodes.Code(status.Convert(err).Code()).String())
			} else {
				require.NoError(t, err)
				got.EventMetadata = nil
				got.AuditContext = nil
				test.CheckProtobufs(t, tt.want, &got, test.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
