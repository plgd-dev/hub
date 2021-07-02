package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/google/go-querystring/query"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	testHttp "github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/cloud/test/config"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
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
			name: "all devices",
			args: args{
				req: &pb.GetDevicesRequest{},
			},
			want: []*pb.Device{
				{
					Types:      []string{"oic.d.cloudDevice", "oic.wk.d"},
					Interfaces: []string{"oic.if.r", "oic.if.baseline"},
					Id:         deviceID,
					Name:       test.TestDeviceName,
					Metadata: &pb.Device_Metadata{
						Status: &commands.ConnectionStatus{
							Value: commands.ConnectionStatus_ONLINE,
						},
					},
				},
			},
		},
		{
			name: "offline devices",
			args: args{
				req: &pb.GetDevicesRequest{
					StatusFilter: []pb.GetDevicesRequest_Status{pb.GetDevicesRequest_OFFLINE},
				},
			},
			want: []*pb.Device{},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := testHttp.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	log.Setup(log.Config{Debug: true})
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type Options struct {
				TypeFilter      []string                      `url:"typeFilter,omitempty"`
				StatusFilter    []pb.GetDevicesRequest_Status `url:"status,omitempty"`
				DeviceIdsFilter []string                      `url:"deviceIdsFilter,omitempty"`
			}
			opt := Options{
				TypeFilter:      tt.args.req.TypeFilter,
				StatusFilter:    tt.args.req.StatusFilter,
				DeviceIdsFilter: tt.args.req.DeviceIdsFilter,
			}
			v, err := query.Values(opt)
			require.NoError(t, err)
			url := fmt.Sprintf("https://%v/"+uri.Devices+"/", config.HTTP_GW_HOST)
			val := v.Encode()
			if val != "" {
				url = url + "?" + val
			}

			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)
			request.Header.Add("Authorization", fmt.Sprintf("bearer %s", token))
			trans := http.DefaultTransport.(*http.Transport).Clone()
			trans.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
			c := http.Client{
				Transport: trans,
			}
			resp, err := c.Do(request)
			require.NoError(t, err)
			defer resp.Body.Close()

			marshaler := runtime.JSONPb{}
			decoder := marshaler.NewDecoder(resp.Body)
			devices := make([]*pb.Device, 0, 1)
			for {
				var dev pb.Device
				err = Unmarshal(resp.StatusCode, decoder, &dev)
				if err == io.EOF {
					break
				}
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.NotEmpty(t, dev.ProtocolIndependentId)
				dev.ProtocolIndependentId = ""
				dev.Metadata.Status.ValidUntil = 0
				devices = append(devices, &dev)
			}
			test.CheckProtobufs(t, tt.want, devices, test.RequireToCheckFunc(require.Equal))
		})
	}
}
