package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	c2cConnectorTest "github.com/plgd-dev/hub/v2/cloud2cloud-connector/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	raEvents "github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func testRequestHandlerGetResourceFromDevice(t *testing.T, events store.Events) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	const switchID = "1"
	type args struct {
		req *pb.GetResourceFromDeviceRequest
	}
	tests := []struct {
		name            string
		args            args
		want            *raEvents.ResourceRetrieved
		wantContentType string
		wantErr         bool
	}{
		{
			name: "valid /light/1",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
				},
			},
			wantContentType: message.AppOcfCbor.String(),
			want: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "", map[string]interface{}{
				"name":  "Light",
				"power": uint64(0),
				"state": false,
			}),
		},
		{
			name: "valid /switches",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesHref),
				},
			},
			wantContentType: message.AppOcfCbor.String(),
			want: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, "", []map[interface{}]interface{}{
				{
					"href": test.TestResourceSwitchesInstanceHref(switchID),
					"if":   []interface{}{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
					"p":    map[interface{}]interface{}{"bm": int64(schema.Discoverable | schema.Observable)},
					"rel":  []interface{}{"hosts"},
					"rt":   []interface{}{types.BINARY_SWITCH},
				},
			}),
		},
		{
			name: "valid /oic/d",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
				},
			},
			wantContentType: message.AppOcfCbor.String(),
			want: pbTest.MakeResourceRetrieved(t, deviceID, device.ResourceURI, test.TestResourceDeviceResourceTypes, "", map[string]interface{}{
				"di":   deviceID,
				"dmv":  "ocf.res.1.3.0",
				"icv":  "ocf.2.0.5",
				"n":    test.TestDeviceName,
				"dmno": test.TestDeviceModelNumber,
				"sv":   test.TestDeviceSoftwareVersion,
			}),
		},
		{
			name: "invalid Href",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/unknown"),
				},
			},
			wantErr: true,
		},
	}

	const timeoutWithPull = config.TEST_TIMEOUT + time.Second*10 // longer timeout is needed because of the 10s sleep in SetUpClouds
	ctx, cancel := context.WithTimeout(context.Background(), timeoutWithPull)
	defer cancel()
	tearDown := c2cConnectorTest.SetUpClouds(ctx, t, deviceID, events, switchID)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetAccessToken(t, c2cConnectorTest.OAUTH_HOST, oauthTest.ClientTest, nil))

	conn, err := grpc.NewClient(c2cConnectorTest.GRPC_GATEWAY_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.GetResourceFromDevice(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantContentType, got.GetData().GetContent().GetContentType())
			pbTest.CmpResourceRetrieved(t, tt.want, got.GetData())
		})
	}
}

func TestRequestHandlerGetResourceFromDevice(t *testing.T) {
	type args struct {
		events store.Events
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "full pulling",
		},
		{
			name: "full events",
			args: args{
				events: store.Events{
					Devices:  events.AllDevicesEvents,
					Device:   events.AllDeviceEvents,
					Resource: events.AllResourceEvents,
				},
			},
		},
		{
			name: "resource events + device,devices pulling",
			args: args{
				events: store.Events{
					Resource: events.AllResourceEvents,
				},
			},
		},
		{
			name: "resource, device events + devices pulling",
			args: args{
				events: store.Events{
					Device:   events.AllDeviceEvents,
					Resource: events.AllResourceEvents,
				},
			},
		},
		{
			name: "device, devices events + resource pulling",
			args: args{
				events: store.Events{
					Devices: events.AllDevicesEvents,
					Device:  events.AllDeviceEvents,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRequestHandlerGetResourceFromDevice(t, tt.args.events)
		})
	}
}
