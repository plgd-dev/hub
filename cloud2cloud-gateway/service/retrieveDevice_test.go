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
	c2cService "github.com/plgd-dev/hub/cloud2cloud-gateway/service"
	c2cTest "github.com/plgd-dev/hub/cloud2cloud-gateway/test"
	"github.com/plgd-dev/hub/cloud2cloud-gateway/uri"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	testHttp "github.com/plgd-dev/hub/test/http"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type sortLinksByHref []interface{}

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

func cleanUpRepresentation(v interface{}) interface{} {
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

		rep, ok := li["rep"].(map[interface{}]interface{})
		if !ok {
			continue
		}
		delete(rep, "pi")
		delete(rep, "piid")
	}
	d["links"] = links
	return v
}

func getResourceBaseRepresentation(deviceID, href string, rt []interface{}, opts map[interface{}]interface{}) map[interface{}]interface{} {
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

func getResourceAllRepresentation(deviceID, href string, rt []interface{}, opts map[interface{}]interface{}) map[interface{}]interface{} {
	rep := map[interface{}]interface{}{
		"if": []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		"rt": rt,
	}

	for k, v := range opts {
		rep[k] = v
	}

	return map[interface{}]interface{}{
		"href": "/" + deviceID + href,
		"rep":  rep,
	}
}

func getDeviceRepresentation(deviceID, deviceName string) map[interface{}]interface{} {
	return map[interface{}]interface{}{
		"di":   deviceID,
		"dmn":  []interface{}{},
		"dmno": "",
		"if":   []interface{}{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
		"n":    deviceName,
		"rt":   []interface{}{types.DEVICE_CLOUD, device.ResourceType},
	}
}

func getDevicesBaseRepresentation(deviceID, deviceName string) interface{} {
	return cleanUpRepresentation(map[interface{}]interface{}{
		"device": getDeviceRepresentation(deviceID, deviceName),
		"links": []interface{}{
			getResourceBaseRepresentation(deviceID, configuration.ResourceURI, []interface{}{configuration.ResourceType}, nil),
			getResourceBaseRepresentation(deviceID, test.TestResourceLightInstanceHref("1"), []interface{}{types.CORE_LIGHT}, nil),
			getResourceBaseRepresentation(deviceID, device.ResourceURI, []interface{}{types.DEVICE_CLOUD, device.ResourceType},
				map[interface{}]interface{}{
					"if": []interface{}{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
				}),
			getResourceBaseRepresentation(deviceID, test.TestResourceSwitchesHref, []interface{}{collection.ResourceType},
				map[interface{}]interface{}{
					"if": []interface{}{interfaces.OC_IF_LL, interfaces.OC_IF_CREATE, interfaces.OC_IF_B, interfaces.OC_IF_BASELINE},
				}),
			getResourceBaseRepresentation(deviceID, platform.ResourceURI, []interface{}{platform.ResourceType},
				map[interface{}]interface{}{
					"if": []interface{}{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
				}),
		},
		"status": "online",
	})
}

func getDevicesAllRepresentation(deviceID, deviceName string) interface{} {
	return cleanUpRepresentation(map[interface{}]interface{}{
		"device": getDeviceRepresentation(deviceID, deviceName),
		"links": []interface{}{
			getResourceAllRepresentation(deviceID, configuration.ResourceURI, []interface{}{configuration.ResourceType},
				map[interface{}]interface{}{
					"n": deviceName,
				}),
			getResourceAllRepresentation(deviceID, device.ResourceURI, []interface{}{types.DEVICE_CLOUD, device.ResourceType},
				map[interface{}]interface{}{
					"di":  deviceID,
					"if":  []interface{}{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
					"dmv": "ocf.res.1.3.0",
					"icv": "ocf.2.0.5",
					"n":   deviceName,
				}),
			getResourceAllRepresentation(deviceID, platform.ResourceURI, []interface{}{platform.ResourceType},
				map[interface{}]interface{}{
					"if":   []interface{}{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
					"mnmn": "ocfcloud.com",
				}),
			getResourceAllRepresentation(deviceID, test.TestResourceLightInstanceHref("1"), []interface{}{types.CORE_LIGHT},
				map[interface{}]interface{}{
					"name":  "Light",
					"power": uint64(0),
					"state": false,
				}),
			getResourceAllRepresentation(deviceID, test.TestResourceSwitchesHref, []interface{}{collection.ResourceType},
				map[interface{}]interface{}{
					"if":                        []interface{}{interfaces.OC_IF_LL, interfaces.OC_IF_CREATE, interfaces.OC_IF_B, interfaces.OC_IF_BASELINE},
					"links":                     []interface{}{},
					"rts":                       []interface{}{"oic.r.switch.binary"},
					"rts-m":                     []interface{}{"oic.r.switch.binary"},
					"x.org.openconnectivity.bl": uint64(94),
				}),
		},
		"status": "online",
	})
}

func TestRequestHandlerRetrieveDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	const deviceIDNotFound = "00010000-0000-0000-0000-000000000001"

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	const textPlain = "text/plain"
	type args struct {
		uri          string
		accept       string
		token        string
		contentQuery string
	}
	tests := []struct {
		name            string
		args            args
		wantContentType string
		wantCode        int
		want            interface{}
	}{
		{
			name: "missing token",
			args: args{
				uri: c2cTest.C2CURI(uri.Devices) + "/" + deviceID,
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: token contains an invalid number of segments",
		},
		{
			name: "expired token",
			args: args{
				uri:   c2cTest.C2CURI(uri.Devices) + "/" + deviceID,
				token: oauthTest.GetServiceToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTestExpired),
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: token is expired",
		},
		{
			name: "notFound",
			args: args{
				uri:    c2cTest.C2CURI(uri.Devices) + "/" + deviceIDNotFound,
				accept: message.AppJSON.String(),
				token:  token,
			},
			wantCode:        http.StatusNotFound,
			wantContentType: textPlain,
			want:            "cannot retrieve device: cannot retrieve device(" + deviceIDNotFound + ") [base]: rpc error: code = NotFound desc = cannot get devices: not found",
		},
		{
			name: "invalid accept",
			args: args{
				uri:    c2cTest.C2CURI(uri.Devices) + "/" + deviceID,
				accept: "application/invalid",
				token:  token,
			},
			wantCode:        http.StatusBadRequest,
			wantContentType: textPlain,
			want:            "cannot retrieve device: invalid accept header([application/invalid])",
		},
		{
			name: "invalid contentQuery",
			args: args{
				uri:          c2cTest.C2CURI(uri.Devices) + "/" + deviceID,
				accept:       message.AppJSON.String(),
				token:        token,
				contentQuery: "invalid",
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: inaccessible uri",
		},
		{
			name: "JSON: " + uri.Devices + "/" + deviceID,
			args: args{
				uri:    c2cTest.C2CURI(uri.Devices) + "/" + deviceID,
				accept: message.AppJSON.String(),
				token:  token,
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want:            getDevicesBaseRepresentation(deviceID, test.TestDeviceName),
		},
		{
			name: "CBOR: " + uri.Devices + "/" + deviceID,
			args: args{
				uri:    c2cTest.C2CURI(uri.Devices) + "/" + deviceID,
				accept: message.AppOcfCbor.String(),
				token:  token,
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppOcfCbor.String(),
			want:            getDevicesBaseRepresentation(deviceID, test.TestDeviceName),
		},
		{
			name: "JSON: " + uri.Devices + "//" + deviceID + "/",
			args: args{
				uri:    c2cTest.C2CURI(uri.Devices) + "//" + deviceID + "/",
				accept: message.AppJSON.String(),
				token:  token,
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want:            getDevicesBaseRepresentation(deviceID, test.TestDeviceName),
		},
		{
			name: "JSON: " + uri.Devices + "/" + deviceID + "?" + c2cService.ContentQuery + "=" + c2cService.ContentQueryAllValue,
			args: args{
				uri:          c2cTest.C2CURI(uri.Devices) + "/" + deviceID,
				accept:       message.AppJSON.String(),
				token:        token,
				contentQuery: c2cService.ContentQueryAllValue,
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want:            getDevicesAllRepresentation(deviceID, test.TestDeviceName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := testHttp.NewHTTPRequest(http.MethodGet, tt.args.uri, nil).Accept(tt.args.accept).AuthToken(tt.args.token)
			if tt.args.contentQuery != "" {
				rb.AddContentQuery(tt.args.contentQuery)
			}
			resp := testHttp.DoHTTPRequest(t, rb.Build(ctx, t))
			assert.Equal(t, tt.wantCode, resp.StatusCode)
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantContentType, resp.Header.Get("Content-Type"))

			got := testHttp.ReadHTTPResponse(t, resp.Body, tt.wantContentType)
			if tt.wantContentType == textPlain {
				require.Contains(t, got, tt.want)
				return
			}
			require.Equal(t, tt.want, cleanUpRepresentation(got))
		})
	}
}
