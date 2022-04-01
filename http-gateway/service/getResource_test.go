package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"testing"

	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func newBool(v bool) *bool {
	return &v
}

func TestRequestHandlerGetResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		accept            string
		deviceID          string
		resourceHref      string
		resourceInterface string
		shadow            *bool
	}
	tests := []struct {
		name string
		args args
		want *events.ResourceRetrieved
	}{
		{
			name: "json: get from resource shadow",
			args: args{
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
			},
			want: &events.ResourceRetrieved{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     test.TestResourceLightInstanceHref("1"),
				},
				Status:       commands.Status_OK,
				Content:      &commands.Content{}, // content is encoded as json
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
			},
		},
		{
			name: "jsonpb: get from resource shadow",
			args: args{
				accept:       uri.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
			},
			want: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceLightInstanceHref("1"), "",
				map[string]interface{}{
					"state": false,
					"power": uint64(0),
					"name":  "Light",
				}),
		},
		{
			name: "jsonpb: get from device with interface",
			args: args{
				accept:            uri.ApplicationProtoJsonContentType,
				deviceID:          deviceID,
				resourceHref:      test.TestResourceLightInstanceHref("1"),
				resourceInterface: interfaces.OC_IF_BASELINE,
			},
			want: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceLightInstanceHref("1"), "",
				map[string]interface{}{
					"state": false,
					"power": uint64(0),
					"name":  "Light",
					"if":    []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
					"rt":    []interface{}{types.CORE_LIGHT},
				},
			),
		},
		{
			name: "jsonpb: get from device with disabled shadow",
			args: args{
				accept:       uri.ApplicationProtoJsonContentType,
				deviceID:     deviceID,
				resourceHref: test.TestResourceLightInstanceHref("1"),
				shadow:       newBool(false),
			},
			want: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceLightInstanceHref("1"), "",
				map[string]interface{}{
					"state": false,
					"power": uint64(0),
					"name":  "Light",
				},
			),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodGet, uri.AliasDeviceResource, nil).AuthToken(token).Accept(tt.args.accept)
			rb.DeviceId(tt.args.deviceID).ResourceHref(tt.args.resourceHref).ResourceInterface(tt.args.resourceInterface)
			if tt.args.shadow != nil {
				rb.Shadow(*tt.args.shadow)
			}
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()

			values := make([]*events.ResourceRetrieved, 0, 1)
			for {
				var value pb.GetResourceFromDeviceResponse
				err = httpgwTest.Unmarshal(resp.StatusCode, resp.Body, &value)
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
				require.NotEmpty(t, value.GetData())
				values = append(values, value.GetData())
			}
			require.Len(t, values, 1)
			pbTest.CmpResourceRetrieved(t, tt.want, values[0])
		})
	}
}
