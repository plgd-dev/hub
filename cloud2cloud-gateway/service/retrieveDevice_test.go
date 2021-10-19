package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"sort"
	"testing"

	"github.com/plgd-dev/device/schema/collection"
	"github.com/plgd-dev/device/schema/configuration"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/schema/platform"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/cloud2cloud-gateway/uri"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	testHttp "github.com/plgd-dev/hub/test/http"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type sortLinksByHref []interface{}

const DeviceIDNotFound = "00010000-0000-0000-0000-000000000001"

func (a sortLinksByHref) Len() int      { return len(a) }
func (a sortLinksByHref) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortLinksByHref) Less(i, j int) bool {
	e1 := a[i].(map[interface{}]interface{})
	e2 := a[j].(map[interface{}]interface{})
	return e1["href"].(string) < e2["href"].(string)
}

func sortLinks(s []interface{}) []interface{} {
	v := sortLinksByHref(s)
	sort.Sort(v)
	return v
}

func cleanUp(v interface{}) interface{} {
	d, ok := v.(map[interface{}]interface{})
	if !ok {
		return v
	}
	device, ok := d["device"].(map[interface{}]interface{})
	if ok {
		delete(device, "piid")
	}
	links, ok := d["links"].([]interface{})
	if !ok {
		return v
	}
	links = sortLinks(links)
	for _, l := range links {
		li, ok := l.(map[interface{}]interface{})
		if !ok {
			continue
		}
		delete(li, "ins")
	}
	d["links"] = links
	return v
}

func getResourceRepresentation(deviceID, href string, rt []interface{}, opts map[interface{}]interface{}) map[interface{}]interface{} {
	res := map[interface{}]interface{}{
		"di":   deviceID,
		"href": "/" + deviceID + href,
		"if":   []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		"p": map[interface{}]interface{}{
			"bm":                 uint64(0x3),
			"port":               uint64(0x0),
			"sec":                false,
			"x.org.iotivity.tcp": uint64(0x0),
			"x.org.iotivity.tls": uint64(0x0),
		},
		"rt": rt,
	}

	for k, v := range opts {
		res[k] = v
	}
	return res
}

func getDeviceAllRepresentation(deviceID, deviceName string) interface{} {
	return cleanUp(map[interface{}]interface{}{
		"device": map[interface{}]interface{}{
			"di":   deviceID,
			"dmn":  []interface{}{},
			"dmno": "",
			"if":   []interface{}{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
			"n":    deviceName,
			"rt":   []interface{}{types.DEVICE_CLOUD, device.ResourceType},
		},
		"links": []interface{}{
			getResourceRepresentation(deviceID, configuration.ResourceURI, []interface{}{configuration.ResourceType}, nil),
			getResourceRepresentation(deviceID, test.TestResourceLightInstanceHref("1"), []interface{}{types.CORE_LIGHT}, nil),
			getResourceRepresentation(deviceID, device.ResourceURI, []interface{}{types.DEVICE_CLOUD, device.ResourceType}, map[interface{}]interface{}{
				"if": []interface{}{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
			}),
			getResourceRepresentation(deviceID, test.TestResourceSwitchesHref, []interface{}{collection.ResourceType}, map[interface{}]interface{}{
				"if": []interface{}{interfaces.OC_IF_LL, interfaces.OC_IF_CREATE, interfaces.OC_IF_B, interfaces.OC_IF_BASELINE},
			}),
			getResourceRepresentation(deviceID, platform.ResourceURI, []interface{}{platform.ResourceType}, map[interface{}]interface{}{
				"if": []interface{}{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
			}),
		},
		"status": "online",
	})
}

func TestRequestHandler_RetrieveDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		uri    string
		accept string
	}
	tests := []struct {
		name            string
		args            args
		wantContentType string
		wantCode        int
		want            interface{}
	}{
		{
			name: "JSON: " + uri.Devices + "/" + deviceID,
			args: args{
				uri:    testHttp.HTTPS_SCHEME + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID,
				accept: message.AppJSON.String(),
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want:            getDeviceAllRepresentation(deviceID, test.TestDeviceName),
		},
		{
			name: "CBOR: " + uri.Devices + "/" + deviceID,
			args: args{
				uri:    testHttp.HTTPS_SCHEME + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID,
				accept: message.AppOcfCbor.String(),
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppOcfCbor.String(),
			want:            getDeviceAllRepresentation(deviceID, test.TestDeviceName),
		},
		{
			name: "notFound",
			args: args{
				uri:    testHttp.HTTPS_SCHEME + testCfg.C2C_GW_HOST + uri.Devices + "/" + DeviceIDNotFound,
				accept: message.AppJSON.String(),
			},
			wantCode:        http.StatusNotFound,
			wantContentType: "text/plain",
			want:            "cannot retrieve device: cannot retrieve device(" + DeviceIDNotFound + ") [base]: rpc error: code = NotFound desc = cannot get devices: not found",
		},
		{
			name: "invalidAccept",
			args: args{
				uri:    testHttp.HTTPS_SCHEME + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID,
				accept: "application/invalid",
			},
			wantCode:        http.StatusBadRequest,
			wantContentType: "text/plain",
			want:            "cannot retrieve device: invalid accept header([application/invalid])",
		},
		{
			name: "JSON: " + uri.Devices + "//" + deviceID + "/",
			args: args{
				uri:    testHttp.HTTPS_SCHEME + testCfg.C2C_GW_HOST + uri.Devices + "//" + deviceID + "/",
				accept: message.AppJSON.String(),
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want:            getDeviceAllRepresentation(deviceID, test.TestDeviceName),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testHttp.NewHTTPRequest(http.MethodGet, tt.args.uri, nil).Accept(tt.args.accept).Build(ctx, t)
			resp := testHttp.DoHTTPRequest(t, req)
			assert.Equal(t, tt.wantCode, resp.StatusCode)
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantContentType, resp.Header.Get("Content-Type"))
			if tt.want != nil {
				got := testHttp.ReadHTTPResponse(t, resp.Body, tt.wantContentType)
				cleanUp(got)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
