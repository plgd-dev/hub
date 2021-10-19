package service_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/plgd-dev/device/schema/collection"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/cloud2cloud-connector/store"
	c2cConnectorTest "github.com/plgd-dev/hub/cloud2cloud-connector/test"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func testRequestHandlerGetResourceFromDevice(t *testing.T, events store.Events) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.GetResourceFromDeviceRequest
	}
	tests := []struct {
		name            string
		args            args
		want            map[string]interface{}
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
			want: map[string]interface{}{
				"name":  "Light",
				"power": uint64(0),
				"state": false,
				"if":    []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
				"rt":    []interface{}{types.CORE_LIGHT},
			},
		},
		{
			name: "valid /switches",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesHref),
				},
			},
			wantContentType: message.AppOcfCbor.String(),
			want: map[string]interface{}{
				"if":                        []interface{}{interfaces.OC_IF_LL, interfaces.OC_IF_CREATE, interfaces.OC_IF_B, interfaces.OC_IF_BASELINE},
				"links":                     []interface{}{},
				"rt":                        []interface{}{collection.ResourceType},
				"rts":                       []interface{}{types.BINARY_SWITCH},
				"rts-m":                     []interface{}{types.BINARY_SWITCH},
				"x.org.openconnectivity.bl": uint64(94),
			},
		},
		{
			name: "valid /oic/d",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
				},
			},
			wantContentType: message.AppOcfCbor.String(),
			want: map[string]interface{}{
				"di":  deviceID,
				"dmv": "ocf.res.1.3.0",
				"icv": "ocf.2.0.5",
				"n":   test.TestDeviceName,
				"if":  []interface{}{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
				"rt":  []interface{}{types.DEVICE_CLOUD, device.ResourceType},
			},
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

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := c2cConnectorTest.SetUpClouds(ctx, t, deviceID, events)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	conn, err := grpc.Dial(c2cConnectorTest.GRPC_GATEWAY_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
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
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantContentType, got.GetData().GetContent().GetContentType())
				var d map[string]interface{}
				err := cbor.Decode(got.GetData().GetContent().GetData(), &d)
				require.NoError(t, err)
				delete(d, "piid")
				assert.Equal(t, tt.want, d)
			}
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
