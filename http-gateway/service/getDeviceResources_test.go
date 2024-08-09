package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getResourceChanged(t *testing.T, deviceID string, href string) *events.ResourceChanged {
	for _, l := range test.GetAllBackendResourceRepresentations(t, deviceID, test.TestDeviceName) {
		rid := commands.ResourceIdFromString(l.Href)
		if rid.GetHref() == href {
			return pbTest.MakeResourceChanged(t, deviceID, rid.GetHref(), l.ResourceTypes, "", l.Representation)
		}
	}
	return nil
}

func makePlatformResourceChanged(t *testing.T, deviceID string) *events.ResourceChanged {
	return getResourceChanged(t, deviceID, platform.ResourceURI)
}

func makeCloudDeviceResourceChanged(t *testing.T, deviceID string) *events.ResourceChanged {
	return getResourceChanged(t, deviceID, device.ResourceURI)
}

func getResourceType(href string) []string {
	for _, l := range test.GetAllBackendResourceLinks() {
		if l.Href == href {
			return l.ResourceTypes
		}
	}
	return nil
}

func getResources(t *testing.T, deviceID, deviceName, switchID string) []*pb.Resource {
	data := test.GetAllBackendResourceRepresentations(t, deviceID, deviceName)
	resources := make([]*pb.Resource, 0, len(data))
	for _, res := range data {
		rid := commands.ResourceIdFromString(res.Href) // validate
		if rid.GetHref() == test.TestResourceSwitchesHref {
			resources = append(resources, &pb.Resource{
				Types: getResourceType(rid.GetHref()),
				Data: pbTest.MakeResourceChanged(t, deviceID, rid.GetHref(), res.ResourceTypes, "", []interface{}{
					map[string]interface{}{
						"href": test.TestResourceSwitchesInstanceHref(switchID),
						"if":   []string{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
						"p": map[string]interface{}{
							"bm": uint64(schema.Discoverable | schema.Observable),
						},
						"rel": []interface{}{"hosts"},
						"rt":  []interface{}{types.BINARY_SWITCH},
					},
				}),
			})
		} else {
			resources = append(resources, &pb.Resource{
				Types: getResourceType(rid.GetHref()),
				Data:  pbTest.MakeResourceChanged(t, deviceID, rid.GetHref(), res.ResourceTypes, "", res.Representation),
			})
		}
	}
	resources = append(resources, &pb.Resource{
		Types: []string{types.BINARY_SWITCH},
		Data:  pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesInstanceHref(switchID), test.TestResourceSwitchesInstanceResourceTypes, "", test.SwitchResourceRepresentation{}),
	})
	return resources
}

func TestRequestHandlerGetDeviceResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	const switchID = "1"
	type args struct {
		deviceID   string
		typeFilter []string
		accept     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.Resource
	}{
		{
			name: "get resource of " + deviceID,
			args: args{
				deviceID: deviceID,
				accept:   pkgHttp.ApplicationProtoJsonContentType,
			},
			want: getResources(t, deviceID, test.TestDeviceName, switchID),
		},
		{
			name: "get oic.wk.d and oic.wk.p of " + deviceID,
			args: args{
				deviceID:   deviceID,
				typeFilter: []string{device.ResourceType, platform.ResourceType},
				accept:     pkgHttp.ApplicationProtoJsonContentType,
			},
			want: []*pb.Resource{
				{
					Types: []string{platform.ResourceType},
					Data:  makePlatformResourceChanged(t, deviceID),
				},
				{
					Types: []string{types.DEVICE_CLOUD, device.ResourceType},
					Data:  makeCloudDeviceResourceChanged(t, deviceID),
				},
			},
		},
		{
			name: "not found",
			args: args{
				deviceID: "notFound",
				accept:   pkgHttp.ApplicationProtoJsonContentType,
			},
			wantErr: true,
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

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	defer func() {
		_ = conn.Close()
	}()
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)
	time.Sleep(time.Millisecond * 200)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodGet, uri.AliasDeviceResources, nil).Accept(tt.args.accept).AuthToken(token)
			rb.DeviceId(tt.args.deviceID).AddTypeFilter(tt.args.typeFilter)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()

			values := make([]*pb.Resource, 0, 1)
			for {
				var value pb.Resource
				err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &value)
				if errors.Is(err, io.EOF) {
					break
				}
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				values = append(values, &value)
			}
			pbTest.CmpResourceValues(t, tt.want, values)
		})
	}
}
