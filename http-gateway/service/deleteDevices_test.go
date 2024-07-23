package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerDeleteDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
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

	type args struct {
		deviceID string
	}
	tests := []struct {
		name         string
		args         args
		want         *pb.DeleteDevicesResponse
		wantHTTPCode int
	}{
		{
			name: "not found",
			args: args{
				deviceID: test.GenerateDeviceIDbyIdx(0),
			},
			want: &pb.DeleteDevicesResponse{
				DeviceIds: nil,
			},
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "all owned",
			args: args{
				deviceID: "",
			},
			want: &pb.DeleteDevicesResponse{
				DeviceIds: []string{deviceID},
			},
			wantHTTPCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := uri.Devices
			if len(tt.args.deviceID) != 0 {
				url = uri.AliasDevice + "/"
			}
			req := httpgwTest.NewRequest(http.MethodDelete, url, nil).AuthToken(token).DeviceId(tt.args.deviceID).Build()
			resp := httpgwTest.HTTPDo(t, req)
			defer func() {
				_ = resp.Body.Close()
			}()
			assert.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got pb.DeleteDevicesResponse
			err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &got)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetDeviceIds(), got.GetDeviceIds())
		})
	}
}
