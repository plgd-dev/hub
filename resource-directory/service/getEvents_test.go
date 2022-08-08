package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/configuration"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/schema/platform"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getOnboardEventForResource(t *testing.T, deviceID, href string) interface{} {
	if href == platform.ResourceURI {
		return pbTest.MakeResourceChanged(t, deviceID, platform.ResourceURI, "",
			map[string]interface{}{
				"mnmn": "ocfcloud.com",
			})
	}

	if href == device.ResourceURI {
		return pbTest.MakeResourceChanged(t, deviceID, device.ResourceURI, "",
			map[string]interface{}{
				"di":  deviceID,
				"dmv": "ocf.res.1.3.0",
				"icv": "ocf.2.0.5",
				"n":   test.TestDeviceName,
			})
	}

	if href == configuration.ResourceURI {
		return pbTest.MakeResourceChanged(t, deviceID, configuration.ResourceURI, "",
			map[string]interface{}{
				"n": test.TestDeviceName,
			})
	}

	if href == test.TestResourceLightInstanceHref("1") {
		return pbTest.MakeResourceChanged(t, deviceID, test.TestResourceLightInstanceHref("1"), "",
			map[string]interface{}{
				"name":  "Light",
				"power": uint64(0),
				"state": false,
			})
	}

	if href == test.TestResourceSwitchesHref {
		return pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesHref, "",
			[]map[string]interface{}{})
	}

	return nil
}

func getAllOnboardEvents(t *testing.T, deviceID string, links []schema.ResourceLink) []interface{} {
	expectedDMU := pbTest.MakeDeviceMetadataUpdated(deviceID, commands.ShadowSynchronization_UNSET, "")
	expectedRLP := pbTest.MakeResourceLinksPublished(deviceID, test.ResourceLinksToResources(deviceID, links), "")
	expectedRCP := getOnboardEventForResource(t, deviceID, platform.ResourceURI)
	expectedRCD := getOnboardEventForResource(t, deviceID, device.ResourceURI)
	expectedRCC := getOnboardEventForResource(t, deviceID, configuration.ResourceURI)
	expectedRCL := getOnboardEventForResource(t, deviceID, test.TestResourceLightInstanceHref("1"))
	expectedRCS := getOnboardEventForResource(t, deviceID, test.TestResourceSwitchesHref)
	return []interface{}{
		expectedDMU,
		expectedRLP,
		expectedRCP,
		expectedRCD,
		expectedRCC,
		expectedRCL,
		expectedRCS,
	}
}

func waitAndCheckEvents(t *testing.T, client pb.GrpcGateway_GetEventsClient, expected []interface{}) {
	getEvents := make([]*pb.Event, 0, 8)
	for {
		value, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		data := pbTest.GetWrappedEvent(value)
		require.NotNil(t, data)
		gotEv := pbTest.ToEvent(data)
		require.NotNil(t, gotEv)
		getEvents = append(getEvents, gotEv)
	}

	expectedEvents := map[string]*pb.Event{}
	for _, expEv := range expected {
		ev := pbTest.ToEvent(expEv)
		require.NotNil(t, ev)
		expectedEvents[pbTest.GetEventID(ev)] = ev
	}

	for _, ev := range getEvents {
		expEv, ok := expectedEvents[pbTest.GetEventID(ev)]
		if !ok {
			assert.Fail(t, "unexpected event", "event: %v", ev)
			continue
		}
		pbTest.CmpEvent(t, expEv, ev, "")
		delete(expectedEvents, pbTest.GetEventID(ev))
	}
	require.Empty(t, expectedEvents)
}

func TestRequestHandlerGetEventsOnOnboard(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resources := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, resources)
	defer shutdownDevSim()

	client, err := c.GetEvents(ctx, &pb.GetEventsRequest{})
	require.NoError(t, err)
	defer func() {
		_ = client.CloseSend()
	}()
	waitAndCheckEvents(t, client, getAllOnboardEvents(t, deviceID, resources))

	for _, res := range resources {
		client, err := c.GetEvents(ctx, &pb.GetEventsRequest{
			ResourceIdFilter: []string{deviceID + res.Href},
		})
		require.NoError(t, err)
		defer func() {
			_ = client.CloseSend()
		}()
		expEv := getOnboardEventForResource(t, deviceID, res.Href)
		waitAndCheckEvents(t, client, []interface{}{expEv})
	}
}

func testRetrieveDeviceEvents(t *testing.T, ctx context.Context, c pb.GrpcGatewayClient, deviceID string, timestampFilter int64) {
	_, err := c.GetResourceFromDevice(ctx, &pb.GetResourceFromDeviceRequest{
		ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
	})
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 200)

	client, err := c.GetEvents(ctx, &pb.GetEventsRequest{
		DeviceIdFilter:  []string{deviceID},
		TimestampFilter: timestampFilter,
	})
	require.NoError(t, err)
	defer func() {
		_ = client.CloseSend()
	}()

	expectedEvents := []interface{}{
		pbTest.MakeResourceRetrievePending(deviceID, device.ResourceURI, ""),
		pbTest.MakeResourceRetrieved(t, deviceID, device.ResourceURI, "",
			map[string]interface{}{
				"di":  deviceID,
				"dmv": "ocf.res.1.3.0",
				"icv": "ocf.2.0.5",
				"n":   test.TestDeviceName,
			},
		),
	}
	waitAndCheckEvents(t, client, expectedEvents)
}

func testUpdateDeviceEvents(t *testing.T, ctx context.Context, c pb.GrpcGatewayClient, deviceID string, timestampFilter int64) {
	_, err := c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
		DeviceId:              deviceID,
		ShadowSynchronization: pb.UpdateDeviceMetadataRequest_ENABLED,
	})
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 200)

	client, err := c.GetEvents(ctx, &pb.GetEventsRequest{
		DeviceIdFilter:  []string{deviceID},
		TimestampFilter: timestampFilter,
	})
	require.NoError(t, err)
	defer func() {
		_ = client.CloseSend()
	}()

	expectedEvents := []interface{}{
		pbTest.MakeDeviceMetadataUpdatePending(deviceID, commands.ShadowSynchronization_ENABLED, ""),
		pbTest.MakeDeviceMetadataUpdated(deviceID, commands.ShadowSynchronization_ENABLED, ""),
	}
	waitAndCheckEvents(t, client, expectedEvents)
}

func testCreateResourceEvents(t *testing.T, ctx context.Context, c pb.GrpcGatewayClient, deviceID string, timestampFilter int64) {
	_, err := c.CreateResource(ctx, &pb.CreateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesHref),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data:        test.EncodeToCbor(t, test.MakeSwitchResourceDefaultData()),
		},
	})
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 200)

	client, err := c.GetEvents(ctx, &pb.GetEventsRequest{
		TimestampFilter: timestampFilter,
	})
	require.NoError(t, err)
	defer func() {
		_ = client.CloseSend()
	}()

	switchID := "1"
	switchResourceLink := test.DefaultSwitchResourceLink(deviceID, switchID)
	rcp := pbTest.MakeResourceCreatePending(t, deviceID, test.TestResourceSwitchesHref, "",
		test.MakeSwitchResourceDefaultData())
	rchangeSwitch := pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesInstanceHref(switchID), "",
		map[string]interface{}{
			"value": false,
		})
	rlp := pbTest.MakeResourceLinksPublished(deviceID, []*commands.Resource{
		{
			DeviceId:      switchResourceLink.DeviceID,
			Href:          switchResourceLink.Href,
			Interfaces:    switchResourceLink.Interfaces,
			ResourceTypes: switchResourceLink.ResourceTypes,
			Policy:        commands.SchemaPolicyToRAPolicy(switchResourceLink.Policy),
		},
	}, "")
	rcreat := pbTest.MakeResourceCreated(t, deviceID, test.TestResourceSwitchesHref, "",
		pbTest.MakeCreateSwitchResourceResponseData(switchID))
	rchangeSwitches := pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesHref, "",
		[]interface{}{
			map[string]interface{}{
				"href": test.TestResourceSwitchesInstanceHref(switchID),
				"if":   []string{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
				"p": map[interface{}]interface{}{
					"bm": uint64(schema.Discoverable | schema.Observable),
				},
				"rel": []string{"hosts"},
				"rt":  []string{types.BINARY_SWITCH},
			},
		})
	expectedEvents := []interface{}{rcp, rlp, rcreat, rchangeSwitch, rchangeSwitches}
	waitAndCheckEvents(t, client, expectedEvents)
}

func testUpdateResourceEvents(t *testing.T, ctx context.Context, c pb.GrpcGatewayClient, deviceID string, timestampFilter int64) {
	switchID := "1"
	switchHref := test.TestResourceSwitchesInstanceHref(switchID)
	switchData := map[string]interface{}{
		"value": true,
	}
	_, err := c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, switchHref),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data:        test.EncodeToCbor(t, switchData),
		},
	})
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 200)

	client, err := c.GetEvents(ctx, &pb.GetEventsRequest{
		DeviceIdFilter:  []string{deviceID},
		TimestampFilter: timestampFilter,
	})
	require.NoError(t, err)
	defer func() {
		_ = client.CloseSend()
	}()

	rup := pbTest.MakeResourceUpdatePending(t, deviceID, switchHref, "", switchData)
	ru := pbTest.MakeResourceUpdated(t, deviceID, switchHref, "", switchData)
	rch := pbTest.MakeResourceChanged(t, deviceID, switchHref, "", switchData)

	expectedEvents := []interface{}{rup, ru, rch}
	waitAndCheckEvents(t, client, expectedEvents)
}

func testDeleteResourceEvents(t *testing.T, ctx context.Context, c pb.GrpcGatewayClient, deviceID string, timestampFilter int64) {
	switchID := "1"
	switchHref := test.TestResourceSwitchesInstanceHref(switchID)
	_, err := c.DeleteResource(ctx, &pb.DeleteResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, switchHref),
	})
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 200)

	client, err := c.GetEvents(ctx, &pb.GetEventsRequest{
		DeviceIdFilter:  []string{deviceID},
		TimestampFilter: timestampFilter,
	})
	require.NoError(t, err)
	defer func() {
		_ = client.CloseSend()
	}()

	rdp := pbTest.MakeResourceDeletePending(deviceID, switchHref, "")
	rd := pbTest.MakeResourceDeleted(t, deviceID, switchHref, "")
	ru := pbTest.MakeResourceLinksUnpublished(deviceID, []string{switchHref}, "")
	rc := pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesHref, "", []interface{}{})

	expectedEvents := []interface{}{rdp, rd, ru, rc}
	waitAndCheckEvents(t, client, expectedEvents)
}

func TestRequestHandlerGetEventsOnCollection(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resources := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, resources)
	defer shutdownDevSim()

	// Retrieve
	time.Sleep(time.Millisecond * 200)
	retrieveResourceFilter := time.Now().UnixNano()
	testRetrieveDeviceEvents(t, ctx, c, deviceID, retrieveResourceFilter)

	// Update device
	time.Sleep(time.Millisecond * 200)
	updateDeviceFilter := time.Now().UnixNano()
	testUpdateDeviceEvents(t, ctx, c, deviceID, updateDeviceFilter)

	// Create resource /switches/1
	time.Sleep(time.Millisecond * 200)
	createResourceFilter := time.Now().UnixNano()
	testCreateResourceEvents(t, ctx, c, deviceID, createResourceFilter)

	// Update resource /switches/1
	time.Sleep(time.Millisecond * 200)
	updateResourceFilter := time.Now().UnixNano()
	testUpdateResourceEvents(t, ctx, c, deviceID, updateResourceFilter)

	// Delete resource /switches/1
	time.Sleep(time.Millisecond * 200)
	deleteResourceFilter := time.Now().UnixNano()
	testDeleteResourceEvents(t, ctx, c, deviceID, deleteResourceFilter)
}
