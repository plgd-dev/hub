package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"sort"
	"testing"

	"github.com/google/go-querystring/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/cloud/v2/http-gateway/test"
	"github.com/plgd-dev/cloud/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/events"
	"github.com/plgd-dev/cloud/v2/test"
	"github.com/plgd-dev/cloud/v2/test/config"
	oauthTest "github.com/plgd-dev/cloud/v2/test/oauth-server/test"
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
		dataWant := want[idx].Data.GetContent().GetData()
		datagot := got[idx].Data.GetContent().GetData()
		want[idx].Data.Content.Data = nil
		got[idx].Data.Content.Data = nil
		test.CheckProtobufs(t, want[idx], got[idx], test.RequireToCheckFunc(require.Equal))
		w := test.DecodeCbor(t, dataWant)
		g := test.DecodeCbor(t, datagot)
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
			name: "valid",
			args: args{
				req: &pb.GetResourcesRequest{
					ResourceIdFilter: []string{
						commands.NewResourceID(deviceID, "/light/1").ToString(),
					},
				},
				accept: uri.ApplicationProtoJsonContentType,
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
			type Options struct {
				TypeFilter       []string `url:"typeFilter,omitempty"`
				ResourceIdFilter []string `url:"resourceIdFilter,omitempty"`
				DeviceIdFilter   []string `url:"deviceIdFilter,omitempty"`
			}
			opt := Options{
				TypeFilter:       tt.args.req.TypeFilter,
				ResourceIdFilter: tt.args.req.ResourceIdFilter,
				DeviceIdFilter:   tt.args.req.DeviceIdFilter,
			}
			v, err := query.Values(opt)
			require.NoError(t, err)
			request := httpgwTest.NewRequest(http.MethodGet, uri.Resources, nil).AuthToken(token).Accept(tt.args.accept).SetQuery(v.Encode()).Build()
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

			values := make([]*pb.Resource, 0, 1)
			for {
				var value pb.Resource
				err = Unmarshal(resp.StatusCode, resp.Body, &value)
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
