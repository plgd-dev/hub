package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"testing"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/go-coap/v3/message"
	c2cService "github.com/plgd-dev/hub/v2/cloud2cloud-gateway/service"
	c2cTest "github.com/plgd-dev/hub/v2/cloud2cloud-gateway/test"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/uri"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type DevicesBaseRepresentation struct {
	Device device.Device        `json:"device"`
	Links  schema.ResourceLinks `json:"links"`
	Status string               `json:"status"`
}

func GetDeviceResourceRepresentation(deviceID, deviceName string) device.Device {
	d := test.GetDeviceResourceRepresentation(deviceID, deviceName)
	return d
}

func getDevicesBaseRepresentation(deviceID, deviceName, switchID string) DevicesBaseRepresentation {
	links := test.GetAllBackendResourceLinks()
	links = append(links, test.DefaultSwitchResourceLink(deviceID, switchID))
	for i := range links {
		links[i].DeviceID = deviceID
		links[i].Href = "/" + commands.NewResourceID(deviceID, links[i].Href).ToString()
	}
	return DevicesBaseRepresentation{
		Device: GetDeviceResourceRepresentation(deviceID, deviceName),
		Links:  links.Sort(),
		Status: "online",
	}
}

type DevicesAllRepresentation struct {
	Device device.Device                    `json:"device"`
	Links  test.ResourceLinkRepresentations `json:"links"`
	Status string                           `json:"status"`
}

func getDevicesAllRepresentation(t *testing.T, deviceID, deviceName, switchID string) DevicesAllRepresentation {
	links := test.GetAllBackendResourceRepresentations(t, deviceID, deviceName)
	for i := range links {
		if strings.HasSuffix(links[i].Href, test.TestResourceSwitchesHref) {
			l := test.DefaultSwitchResourceLink(deviceID, switchID)
			l.DeviceID = ""
			links[i].Representation = schema.ResourceLinks{l}
			continue
		}
	}
	links = append(links, test.ResourceLinkRepresentation{
		Href:           "/" + commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID)).ToString(),
		Representation: test.SwitchResourceRepresentation{},
	})
	return DevicesAllRepresentation{
		Device: GetDeviceResourceRepresentation(deviceID, deviceName),
		Links:  links.Sort(),
		Status: "online",
	}
}

func TestRequestHandlerRetrieveDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	const deviceIDNotFound = "00010000-0000-0000-0000-000000000001"

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
	const switchID = "1"
	// create subscription to wait resourceChanged event
	subClient, err := c.SubscribeToEvents(ctx)
	require.NoError(t, err)
	err = subClient.Send(&pb.SubscribeToEvents{
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				DeviceIdFilter: []string{deviceID},
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
				},
			},
		},
	})
	require.NoError(t, err)
	resp, err := subClient.Recv()
	require.NoError(t, err)
	require.Equal(t, pb.Event_OperationProcessed_ErrorStatus_OK, resp.GetOperationProcessed().GetErrorStatus().GetCode())
	defer func() {
		err := subClient.CloseSend()
		require.NoError(t, err)
	}()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)
	// wait for resource changed
	for {
		ev, err := subClient.Recv()
		require.NoError(t, err)
		if ev.GetResourceChanged().GetResourceId().GetDeviceId() == deviceID && ev.GetResourceChanged().GetResourceId().GetHref() == test.TestResourceSwitchesInstanceHref(switchID) {
			break
		}
	}

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
			want:            "invalid token: could not parse token: token is malformed: token contains an invalid number of segments",
		},
		{
			name: "expired token",
			args: args{
				uri:   c2cTest.C2CURI(uri.Devices) + "/" + deviceID,
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTestExpired, nil),
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: token has invalid claims: token is expired",
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
			want:            "invalid token: could not parse token: token has invalid claims: inaccessible uri",
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
			want:            getDevicesBaseRepresentation(deviceID, test.TestDeviceName, switchID),
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
			want:            getDevicesBaseRepresentation(deviceID, test.TestDeviceName, switchID),
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
			want:            getDevicesBaseRepresentation(deviceID, test.TestDeviceName, switchID),
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
			want:            getDevicesAllRepresentation(t, deviceID, test.TestDeviceName, switchID),
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
			var got interface{}
			if _, ok := tt.want.(DevicesAllRepresentation); ok {
				d := DevicesAllRepresentation{}
				testHttp.ReadHTTPResponse(t, resp.Body, tt.wantContentType, &d)
				require.NotEmpty(t, d.Device.ProtocolIndependentID)
				d.Device.ProtocolIndependentID = ""
				d.Device.ManufacturerName = nil
				d.Links = d.Links.Sort()
				got = d
			} else if _, ok := tt.want.(DevicesBaseRepresentation); ok {
				d := DevicesBaseRepresentation{}
				testHttp.ReadHTTPResponse(t, resp.Body, tt.wantContentType, &d)
				require.NotEmpty(t, d.Device.ProtocolIndependentID)
				d.Device.ProtocolIndependentID = ""
				d.Device.ManufacturerName = nil
				d.Links = d.Links.Sort()
				got = d
			} else {
				testHttp.ReadHTTPResponse(t, resp.Body, tt.wantContentType, &got)
			}
			if tt.wantContentType == textPlain {
				require.Contains(t, got, tt.want)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}
