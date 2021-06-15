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
	"github.com/plgd-dev/go-coap/v2/message"
)

func cmpResourceValues(t *testing.T, want []*pb.Resource, got []*pb.Resource) {
	require.Len(t, got, len(want))
	for idx := range want {
		dataWant := want[idx].Data.GetContent().GetData()
		datagot := got[idx].Data.GetContent().GetData()
		want[idx].Data.Content.Data = nil
		got[idx].Data.Content.Data = nil
		test.CheckProtobufs(t, want[idx], got[idx], test.RequireToCheckFunc(require.Equal))
		w := test.DecodeCbor(t, dataWant)
		g := test.DecodeCbor(t, datagot)
		require.Equal(t, w, g)
	}
}

func TestRequestHandler_RetrieveResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.RetrieveResourcesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.Resource
	}{
		{
			name: "valid",
			args: args{
				req: &pb.RetrieveResourcesRequest{
					ResourceIdsFilter: []string{
						commands.NewResourceID(deviceID, "/light/1").ToString(),
					},
				},
			},
			want: []*pb.Resource{
				{
					Types: []string{"core.light"},
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: deviceID,
							Href:     "/light/1",
						},
						Content: &commands.Content{
							CoapContentFormat: int32(message.AppOcfCbor),
							ContentType:       message.AppOcfCbor.String(),
							Data: test.EncodeToCbor(t, map[string]interface{}{
								"state": false,
								"power": uint64(0),
								"name":  "Light",
								"if":    []interface{}{"oic.if.rw", "oic.if.baseline"},
								"rt":    []interface{}{"core.light"},
							}),
						},
						Status: commands.Status_OK,
					},
				},
			},
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
				TypeFilter        []string `url:"typeFilter,omitempty"`
				ResourceIdsFilter []string `url:"resourceIdsFilter,omitempty"`
				DeviceIdsFilter   []string `url:"deviceIdsFilter,omitempty"`
			}
			opt := Options{
				TypeFilter:        tt.args.req.TypeFilter,
				ResourceIdsFilter: tt.args.req.ResourceIdsFilter,
				DeviceIdsFilter:   tt.args.req.DeviceIdsFilter,
			}
			v, err := query.Values(opt)
			require.NoError(t, err)
			request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%v/api/v1/resources?%v", config.HTTP_GW_HOST, v.Encode()), nil)
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
			values := make([]*pb.Resource, 0, 1)
			for {
				var value pb.Resource
				err = service.Unmarshal(resp.StatusCode, decoder, &value)
				if err == io.EOF {
					break
				}
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				value.Data.AuditContext = nil
				value.Data.EventMetadata = nil
				values = append(values, &value)
			}
			cmpResourceValues(t, tt.want, values)
		})
	}
}
