package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/go-coap/v3/message"
	coapTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/sdk"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerGetResourceFromDevice(t *testing.T) {
	deviceName := test.TestDeviceName
	deviceID := test.MustFindDeviceByName(deviceName)
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
					ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
					TimeToLive: int64(99 * time.Millisecond),
				},
			},
			wantErr: true,
		},
		{
			name: "valid /light/1",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
					TimeToLive: int64(time.Hour),
				},
			},
			want: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "",
				map[string]interface{}{
					"name":  "Light",
					"power": uint64(0),
					"state": false,
				},
			),
		},
		{
			name: "valid /oic/d",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
					TimeToLive: int64(time.Hour),
				},
			},
			want: pbTest.MakeResourceRetrieved(t, deviceID, device.ResourceURI, test.TestResourceDeviceResourceTypes, "",
				map[string]interface{}{
					"n":    deviceName,
					"di":   deviceID,
					"dmv":  "ocf.res.1.3.0",
					"icv":  "ocf.2.0.5",
					"dmno": test.TestDeviceModelNumber,
					"sv":   test.TestDeviceSoftwareVersion,
				},
			),
		},
		{
			name: "valid /switches",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesHref),
					TimeToLive: int64(time.Hour),
				},
			},
			want: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, "",
				[]map[string]interface{}{
					{
						"href": test.TestResourceSwitchesInstanceHref(switchID),
						"if":   []interface{}{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
						"rt":   []interface{}{types.BINARY_SWITCH},
						"rel":  []interface{}{"hosts"},
						"p": map[string]interface{}{
							"bm": uint64(schema.Discoverable | schema.Observable),
						},
						"eps": []interface{}{},
					},
				},
			),
		},
		{
			name: "valid /switches/1",
			args: args{
				req: &pb.GetResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID)),
					TimeToLive: int64(time.Hour),
				},
			},
			want: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceSwitchesInstanceHref(switchID), test.TestResourceSwitchesInstanceResourceTypes, "",
				map[string]interface{}{
					"value": false,
				},
			),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	coapCfg := coapTest.MakeConfig(t)
	tearDown := service.SetUp(ctx, t, service.WithCOAPGWConfig(coapCfg))
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)
	// for update resource-directory cache
	time.Sleep(time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.GetResourceFromDevice(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			pbTest.CmpResourceRetrieved(t, tt.want, got.GetData())
		})
	}
}

func validateETags(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, deviceID, href string) {
	sdkClient, err := sdk.NewClient()
	require.NoError(t, err)

	defer func() {
		errC := sdkClient.Close(context.Background())
		require.NoError(t, errC)
	}()

	// get resource from device via SDK
	cfg1 := coap.DetailedResponse[interface{}]{}
	err = sdkClient.GetResource(ctx, deviceID, href, &cfg1)
	require.NoError(t, err)

	// get resource from device via HUB
	cfg2, err := c.GetResourceFromDevice(ctx, &pb.GetResourceFromDeviceRequest{
		ResourceId: commands.NewResourceID(deviceID, href),
		TimeToLive: int64(time.Hour),
	})
	require.NoError(t, err)

	checkTag, err := c.GetResourceFromDevice(ctx, &pb.GetResourceFromDeviceRequest{
		ResourceId: commands.NewResourceID(deviceID, href),
		TimeToLive: int64(time.Hour),
		Etag:       [][]byte{cfg2.GetData().GetEtag()},
	})
	require.NoError(t, err)
	require.Equal(t, commands.Status_NOT_MODIFIED, checkTag.GetData().GetStatus())
	require.Empty(t, checkTag.GetData().GetContent().GetData())
	require.Empty(t, checkTag.GetData().GetContent().GetContentType())
	require.Equal(t, int32(-1), checkTag.GetData().GetContent().GetCoapContentFormat())

	// compare SDK and HUB ETags
	require.Equal(t, cfg1.ETag, cfg2.GetData().GetEtag())
	var body2 interface{}
	err = cbor.Decode(cfg2.GetData().GetContent().GetData(), &body2)
	require.NoError(t, err)
	require.Equal(t, cfg1.Body, body2)

	// get resource from device twin
	clients, err := c.GetResources(ctx, &pb.GetResourcesRequest{
		ResourceIdFilter: []*pb.ResourceIdFilter{
			{
				ResourceId: commands.NewResourceID(deviceID, href),
			},
		},
	})
	require.NoError(t, err)

	var etag3 []byte
	var body3 interface{}

	for {
		res, err := clients.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		etag3 = res.GetData().GetEtag()
		err = cbor.Decode(res.GetData().GetContent().GetData(), &body3)
		require.NoError(t, err)
	}

	require.Equal(t, cfg1.Body, body3)
	// compare SDK and DeviceTwin ETags
	require.Equal(t, cfg1.ETag, etag3)
}

func TestRequestHandlerCheckResourceETag(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	coapCfg := coapTest.MakeConfig(t)
	tearDown := service.SetUp(ctx, t, service.WithCOAPGWConfig(coapCfg))
	defer tearDown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	href := test.TestResourceLightInstanceHref("1")
	validateETags(ctx, t, c, deviceID, href)
	v := test.LightResourceRepresentation{Power: 99}
	_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, href),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data:        test.EncodeToCbor(t, v),
		},
	})
	require.NoError(t, err)
	time.Sleep(time.Second)
	validateETags(ctx, t, c, deviceID, href)
	v = test.LightResourceRepresentation{Power: 0}
	_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, href),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data:        test.EncodeToCbor(t, v),
		},
	})
	require.NoError(t, err)
	time.Sleep(time.Second)
	validateETags(ctx, t, c, deviceID, href)
}
