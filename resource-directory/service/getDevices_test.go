package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/cloud/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
)

func TestRequestHandler_GetDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.GetDevicesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.Device
	}{
		{
			name: "valid",
			args: args{
				req: &pb.GetDevicesRequest{},
			},
			want: []*pb.Device{
				{
					Types:      []string{"oic.d.cloudDevice", "oic.wk.d"},
					Interfaces: []string{"oic.if.r", "oic.if.baseline"},
					Id:         deviceID,
					Name:       test.TestDeviceName,
					IsOnline:   true,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetDevices(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				devices := make([]*pb.Device, 0, 1)
				for {
					dev, err := client.Recv()
					if err == io.EOF {
						break
					}
					require.NoError(t, err)
					devices = append(devices, dev)
				}
				require.Equal(t, tt.want, devices)
			}
		})
	}
}
