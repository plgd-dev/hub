package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
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
	"github.com/plgd-dev/go-coap/v2/message"
)

func TestRequestHandler_RetrieveResourceFromDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req pb.RetrieveResourceFromDeviceRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *events.ResourceRetrieved
		wantErr bool
	}{
		{
			name: "valid /light/2",
			args: args{
				req: pb.RetrieveResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/light/2").ToString(),
				},
			},
			want: &events.ResourceRetrieved{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     "/light/2",
				},
				Content: &commands.Content{
					CoapContentFormat: int32(message.AppOcfCbor),
					ContentType:       message.AppOcfCbor.String(),
				},
				Status: commands.Status_OK,
			},
		},
		{
			name: "valid /oic/d",
			args: args{
				req: pb.RetrieveResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/oic/d").ToString(),
				},
			},
			want: &events.ResourceRetrieved{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     "/oic/d",
				},
				Content: &commands.Content{
					CoapContentFormat: int32(message.AppOcfCbor),
					ContentType:       message.AppOcfCbor.String(),
				},
				Status: commands.Status_OK,
			},
		},
		{
			name: "invalid Href",
			args: args{
				req: pb.RetrieveResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/unknown").ToString(),
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

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type Options struct {
				Interface string `url:"resourceInterface,omitempty"`
			}
			opt := Options{
				Interface: tt.args.req.ResourceInterface,
			}
			v, err := query.Values(opt)
			require.NoError(t, err)
			request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%v/api/v1/devices/%v?%v", config.HTTP_GW_HOST, tt.args.req.ResourceId, v.Encode()), nil)
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
			var got events.ResourceRetrieved
			err = service.Unmarshal(resp.StatusCode, decoder, &got)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				got.EventMetadata = nil
				got.AuditContext = nil
				got.Content.Data = nil
				test.CheckProtobufs(t, tt.want, &got, test.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
