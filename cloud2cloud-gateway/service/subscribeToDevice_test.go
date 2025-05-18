package service_test

import (
	"context"
	"crypto/tls"
	"strings"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	c2cEvents "github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	c2cTest "github.com/plgd-dev/hub/v2/cloud2cloud-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func testSubscribeToDeviceDecodeResources(links ...schema.ResourceLinks) []*commands.Resource {
	resources := make([]*commands.Resource, 0)
	for _, link := range links {
		for _, l := range link {
			pl := commands.SchemaResourceLinkToResource(l, time.Time{})
			pl.Href = "/" + strings.Join(strings.Split(pl.GetHref(), "/")[2:], "/")
			resources = append(resources, pl)
		}
	}
	test.CleanUpResourcesArray(resources)
	return resources
}

func TestRequestHandlerSubscribeToDevicePublishedOnly(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
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

	const eventsURI = "/events"
	eventsServer := c2cTest.NewEventsServer(t, eventsURI)
	defer eventsServer.Close(t)
	dataChan := eventsServer.Run(t)

	subscriber := c2cTest.NewC2CSubscriber(eventsServer.GetPort(t), eventsURI, c2cTest.SubscriptionType_Device)
	subID := subscriber.Subscribe(ctx, t, token, deviceID, "", c2cEvents.EventTypes{c2cEvents.EventType_ResourcesPublished})

	ev := <-dataChan
	publishedResources := test.ResourceLinksToResources(deviceID, test.GetAllBackendResourceLinks())
	assert.Equal(t, c2cEvents.EventType_ResourcesPublished, ev.GetHeader().EventType)
	links := ev.GetData().(schema.ResourceLinks)
	resources := testSubscribeToDeviceDecodeResources(links)
	test.CheckProtobufs(t, publishedResources, resources, test.RequireToCheckFunc(require.Equal))

	// no additional messages should be received
	events := c2cTest.WaitForEvents(dataChan, 3*time.Second)
	require.Empty(t, events)

	subscriber.Unsubscribe(ctx, t, token, deviceID, "", subID)
	ev = <-dataChan
	assert.Equal(t, c2cEvents.EventType_SubscriptionCanceled, ev.GetHeader().EventType)
}

func TestRequestHandlerSubscribeToDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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

	const eventsURI = "/events"
	eventsServer := c2cTest.NewEventsServer(t, eventsURI)
	defer eventsServer.Close(t)
	dataChan := eventsServer.Run(t)

	subscriber := c2cTest.NewC2CSubscriber(eventsServer.GetPort(t), eventsURI, c2cTest.SubscriptionType_Device)
	subID := subscriber.Subscribe(ctx, t, token, deviceID, "", c2cEvents.EventTypes{
		c2cEvents.EventType_ResourcesPublished,
		c2cEvents.EventType_ResourcesUnpublished,
	})

	events := c2cTest.WaitForEvents(dataChan, 3*time.Second)
	// we should always receive one Published event and one Unpublished event
	require.Len(t, events, 2)
	for _, ev := range events {
		if ev.GetHeader().EventType == c2cEvents.EventType_ResourcesPublished {
			publishedResources := test.ResourceLinksToResources(deviceID, test.GetAllBackendResourceLinks())
			links := ev.GetData().(schema.ResourceLinks)
			resources := testSubscribeToDeviceDecodeResources(links)
			test.CheckProtobufs(t, publishedResources, resources, test.RequireToCheckFunc(require.Equal))
			continue
		}
		if ev.GetHeader().EventType == c2cEvents.EventType_ResourcesUnpublished {
			links := ev.GetData().(schema.ResourceLinks)
			require.Empty(t, links)
			continue
		}
		require.Failf(t, "unexpected event", "%v", ev.GetHeader().EventType)
	}

	const switchID1 = "1"
	const switchID2 = "2"
	const switchID3 = "3"
	switchIDs := []string{switchID1, switchID2, switchID3}
	switchResources := test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchIDs...)
	publishedSwitches := test.ResourceLinksToResources(deviceID, switchResources)
	events = c2cTest.WaitForEvents(dataChan, 3*time.Second)
	var links schema.ResourceLinks
	for _, ev := range events {
		require.Equal(t, c2cEvents.EventType_ResourcesPublished, ev.GetHeader().EventType)
		links = append(links, ev.GetData().(schema.ResourceLinks)...)
	}
	resources := testSubscribeToDeviceDecodeResources(links)
	test.CheckProtobufs(t, publishedSwitches, resources, test.RequireToCheckFunc(require.Equal))

	_, err = c.DeleteResource(ctx, &pb.DeleteResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID2)),
	})
	require.NoError(t, err)
	ev := <-dataChan
	require.Equal(t, c2cEvents.EventType_ResourcesUnpublished, ev.GetHeader().EventType)
	links = ev.GetData().(schema.ResourceLinks)
	resources = testSubscribeToDeviceDecodeResources(links)
	var unpublishedSwitches []*commands.Resource
	for _, res := range resources {
		if res.GetHref() == test.TestResourceSwitchesInstanceHref(switchID2) {
			unpublishedSwitches = append(unpublishedSwitches, &commands.Resource{
				DeviceId: deviceID,
				Href:     test.TestResourceSwitchesInstanceHref(switchID2),
			})
		}
	}
	test.CheckProtobufs(t, unpublishedSwitches, resources, test.RequireToCheckFunc(require.Equal))

	// no additional messages should be received
	events = c2cTest.WaitForEvents(dataChan, 3*time.Second)
	require.Empty(t, events)

	subscriber.Unsubscribe(ctx, t, token, deviceID, "", subID)
	ev = <-dataChan
	assert.Equal(t, c2cEvents.EventType_SubscriptionCanceled, ev.GetHeader().EventType)
}
