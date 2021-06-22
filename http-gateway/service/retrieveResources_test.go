package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"sort"
	"testing"

	"github.com/google/go-querystring/query"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/service"
	httpgwTest "github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message"
)

type sortResourcesByHref []*pb.Resource

func (a sortResourcesByHref) Len() int      { return len(a) }
func (a sortResourcesByHref) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortResourcesByHref) Less(i, j int) bool {
	return a[i].GetData().GetResourceId().GetHref() < a[j].GetData().GetResourceId().GetHref()
}

func sortResources(s []*pb.Resource) []*pb.Resource {
	v := sortResourcesByHref(s)
	sort.Sort(v)
	return v
}

func cmpResourceValues(t *testing.T, want []*pb.Resource, got []*pb.Resource) {
	require.Len(t, got, len(want))
	sortResources(want)
	sortResources(got)

	for idx := range want {
		wantJSON, err := want[idx].Data.GetContent().ToJSON()
		require.NoError(t, err)
		dataWant := wantJSON.GetData()
		gotJSON, err := got[idx].Data.GetContent().ToJSON()
		require.NoError(t, err)
		datagot := gotJSON.GetData()

		want[idx].Data.Content.Data = nil
		got[idx].Data.Content.Data = nil
		test.CheckProtobufs(t, want[idx], got[idx], test.RequireToCheckFunc(require.Equal))

		w := test.DecodeJson(t, dataWant)
		g := test.DecodeJson(t, datagot)
		if gV, ok := g.(map[interface{}]interface{}); ok {
			delete(gV, "pi")
			delete(gV, "piid")
		}
		require.Equal(t, w, g)

	}
}

func TestRequestHandler_GetResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req    *pb.GetResourcesRequest
		accept string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.Resource
	}{
		{
			name: "valid - without accept",
			args: args{
				req: &pb.GetResourcesRequest{
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
		{
			name: "valid - accept ocf-cbor",
			args: args{
				req: &pb.GetResourcesRequest{
					ResourceIdsFilter: []string{
						commands.NewResourceID(deviceID, "/light/1").ToString(),
					},
				},
				accept: message.AppOcfCbor.String(),
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
		{
			name: "valid - accept json",
			args: args{
				req: &pb.GetResourcesRequest{
					ResourceIdsFilter: []string{
						commands.NewResourceID(deviceID, "/light/1").ToString(),
					},
				},
				accept: message.AppJSON.String(),
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
							CoapContentFormat: int32(message.AppJSON),
							ContentType:       message.AppJSON.String(),
							Data: test.EncodeToJson(t, map[string]interface{}{
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
			request := httpgwTest.NewRequest(http.MethodGet, uri.Resources, nil).AuthToken(token).SetQuery(v.Encode()).AcceptContent(tt.args.accept).Build()
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
