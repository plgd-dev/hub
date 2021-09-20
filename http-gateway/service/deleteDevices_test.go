package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandler_deleteDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
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

	type args struct {
		deviceID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *pb.DeleteDevicesResponse
	}{
		{
			name: "not found",
			args: args{
				deviceID: "notFound",
			},
			want: &pb.DeleteDevicesResponse{
				DeviceIds: nil,
			},
		},
		{
			name: "all owned",
			args: args{
				deviceID: "",
			},
			want: &pb.DeleteDevicesResponse{
				DeviceIds: []string{deviceID},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := uri.Devices
			if len(tt.args.deviceID) != 0 {
				url = uri.AliasDevice + "/"
			}
			request_builder := httpgwTest.NewRequest(http.MethodDelete, url, nil).AuthToken(token)
			request_builder.DeviceId(tt.args.deviceID)
			request := request_builder.Build()
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

			var got pb.DeleteDevicesResponse
			err = Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.want.DeviceIds, got.DeviceIds)
		})
	}
}
