package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/pkg/ocf"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getResourceRetrieved(t *testing.T, deviceID, href string, data interface{}) *events.ResourceRetrieved {
	return &events.ResourceRetrieved{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Content: &commands.Content{
			CoapContentFormat: int32(message.AppOcfCbor),
			ContentType:       message.AppOcfCbor.String(),
			Data:              test.EncodeToCbor(t, data),
		},
		Status: commands.Status_OK,
	}
}

func TestRequestHandlerGetResourceFromDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	switchID := "1"
	type args struct {
		req *pb.GetResourceFromDeviceRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *events.ResourceRetrieved
		wantErr bool
	}{
		{
			name: "invalid Href",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/unknown"),
					TimeToLive: int64(time.Hour),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid timeToLive",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.OCFResourceDeviceHref),
					TimeToLive: int64(99 * time.Millisecond),
				},
			},
			wantErr: true,
		},
		{
			name: "valid /light/1",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightHref),
					TimeToLive: int64(time.Hour),
				},
			},
			want: getResourceRetrieved(t, deviceID, test.TestResourceLightHref, map[string]interface{}{
				"name":  "Light",
				"power": uint64(0),
				"state": false,
			}),
		},
		{
			name: "valid /oic/d",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.OCFResourceDeviceHref),
					TimeToLive: int64(time.Hour),
				},
			},
			want: getResourceRetrieved(t, deviceID, test.OCFResourceDeviceHref, map[string]interface{}{
				"n":   test.TestDeviceName,
				"di":  deviceID,
				"dmv": "ocf.res.1.3.0",
				"icv": "ocf.2.0.5",
			}),
		},
		{
			name: "valid /switches",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesHref),
					TimeToLive: int64(time.Hour),
				},
			},
			want: getResourceRetrieved(t, deviceID, test.TestResourceSwitchesHref, []map[string]interface{}{
				{
					"href": test.TestResourceSwitchesInstanceHref(switchID),
					"if":   []interface{}{ocf.OC_IF_A, ocf.OC_IF_BASELINE},
					"rt":   []interface{}{ocf.OC_RT_RESOURCE_SWITCH},
					"rel":  []interface{}{"hosts"},
					"p": map[string]interface{}{
						"bm": uint64(schema.Discoverable | schema.Observable),
					},
					"eps": []interface{}{},
				},
			}),
		},
		{
			name: "valid /switches/1",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID)),
					TimeToLive: int64(time.Hour),
				},
			},
			want: getResourceRetrieved(t, deviceID, test.TestResourceSwitchesInstanceHref(switchID), map[string]interface{}{
				"value": false,
			}),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.GetResourceFromDevice(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			test.CmpResourceRetrieved(t, tt.want, got.GetData())
		})
	}
}
