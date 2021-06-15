package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/google/go-querystring/query"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/service"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/cloud/test/config"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
)

func cmpDeviceMetadataUpdated(t *testing.T, want []*events.DeviceMetadataUpdated, got []*events.DeviceMetadataUpdated) {
	require.Len(t, got, len(want))
	for idx := range want {
		got[idx].EventMetadata = nil
		got[idx].AuditContext = nil
		if got[idx].GetStatus() != nil {
			got[idx].GetStatus().ValidUntil = 0
		}
		test.CheckProtobufs(t, want[idx], got[idx], test.RequireToCheckFunc(require.Equal))

	}
}

func TestRequestHandler_RetrieveDevicesMetadata(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req pb.RetrieveDevicesMetadataRequest
	}
	tests := []struct {
		name    string
		args    args
		want    []*events.DeviceMetadataUpdated
		wantErr bool
	}{
		{
			name: "all",
			args: args{
				req: pb.RetrieveDevicesMetadataRequest{},
			},
			want: []*events.DeviceMetadataUpdated{
				{
					DeviceId: deviceID,
					Status: &commands.ConnectionStatus{
						Value: commands.ConnectionStatus_ONLINE,
					},
				},
			},
		},
		{
			name: "filter one device",
			args: args{
				req: pb.RetrieveDevicesMetadataRequest{
					DeviceIdsFilter: []string{deviceID},
				},
			},
			want: []*events.DeviceMetadataUpdated{
				{
					DeviceId: deviceID,
					Status: &commands.ConnectionStatus{
						Value: commands.ConnectionStatus_ONLINE,
					},
				},
			},
		},
		{
			name: "filter one device by type",
			args: args{
				req: pb.RetrieveDevicesMetadataRequest{
					TypeFilter: []string{"oic.wk.d"},
				},
			},
			want: []*events.DeviceMetadataUpdated{
				{
					DeviceId: deviceID,
					Status: &commands.ConnectionStatus{
						Value: commands.ConnectionStatus_ONLINE,
					},
				},
			},
		},
		{
			name: "invalid deviceID",
			args: args{
				req: pb.RetrieveDevicesMetadataRequest{
					DeviceIdsFilter: []string{"abc"},
				},
			},
			wantErr: true,
		},
		{
			name: "unknown type",
			args: args{
				req: pb.RetrieveDevicesMetadataRequest{
					TypeFilter: []string{"unknown"},
				},
			},
			wantErr: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := New(t, MakeConfig(t))
	defer shutdownHttp()

	token := oauthTest.GetServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type Options struct {
				TypeFilter      []string `url:"typeFilter,omitempty"`
				DeviceIdsFilter []string `url:"deviceIdsFilter,omitempty"`
			}
			opt := Options{
				TypeFilter:      tt.args.req.TypeFilter,
				DeviceIdsFilter: tt.args.req.DeviceIdsFilter,
			}
			v, err := query.Values(opt)
			require.NoError(t, err)
			request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%v/api/v1/devices/metadata?%v", config.HTTP_GW_HOST, v.Encode()), nil)
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
			values := make([]*events.DeviceMetadataUpdated, 0, 1)
			for {
				var value events.DeviceMetadataUpdated
				err = service.Unmarshal(resp.StatusCode, decoder, &value)
				if err == io.EOF {
					break
				}
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				values = append(values, &value)
			}
			cmpDeviceMetadataUpdated(t, tt.want, values)
		})
	}
}
