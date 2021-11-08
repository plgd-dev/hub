package service_test

import (
	"context"
	"crypto/tls"
	"strings"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/hub/cloud2cloud-connector/events"
	c2cTest "github.com/plgd-dev/hub/cloud2cloud-gateway/test"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func testSubscribeToDeviceDecodeResources(t *testing.T, links ...schema.ResourceLinks) []*commands.Resource {
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
	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()
	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

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

	const eventsURI = "/events"
	eventsServer := c2cTest.NewEventsServer(t, eventsURI)
	defer eventsServer.Close(t)
	dataChan := eventsServer.Run(t)

	subscriber := c2cTest.NewC2CSubscriber(eventsServer.GetPort(t), eventsURI)
	subID := subscriber.Subscribe(t, ctx, token, deviceID, events.EventTypes{events.EventType_ResourcesPublished})

	ev := <-dataChan
	publishedResources := test.ResourceLinksToResources(deviceID, test.TestDevsimResources)
	assert.Equal(t, events.EventType_ResourcesPublished, ev.GetHeader().EventType)
	links := ev.GetData().(schema.ResourceLinks)
	resources := testSubscribeToDeviceDecodeResources(t, links)
	test.CheckProtobufs(t, publishedResources, resources, test.RequireToCheckFunc(require.Equal))

	// no additional messages should be received
	select {
	case ev = <-dataChan:
		require.FailNow(t, "unexpected message received")
	case <-time.After(5 * time.Second):
	}

	subscriber.Unsubscribe(t, ctx, token, deviceID, subID)
	ev = <-dataChan
	assert.Equal(t, events.EventType_SubscriptionCanceled, ev.GetHeader().EventType)
}

func TestRequestHandlerSubscribeToDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()
	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

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

	const eventsURI = "/events"
	eventsServer := c2cTest.NewEventsServer(t, eventsURI)
	defer eventsServer.Close(t)
	dataChan := eventsServer.Run(t)

	subscriber := c2cTest.NewC2CSubscriber(eventsServer.GetPort(t), eventsURI)
	subID := subscriber.Subscribe(t, ctx, token, deviceID, events.EventTypes{events.EventType_ResourcesPublished, events.EventType_ResourcesUnpublished})

	ev := <-dataChan
	publishedResources := test.ResourceLinksToResources(deviceID, test.TestDevsimResources)
	assert.Equal(t, events.EventType_ResourcesPublished, ev.GetHeader().EventType)
	links := ev.GetData().(schema.ResourceLinks)
	resources := testSubscribeToDeviceDecodeResources(t, links)
	test.CheckProtobufs(t, publishedResources, resources, test.RequireToCheckFunc(require.Equal))

	const switchID1 = "1"
	const switchID2 = "2"
	const switchID3 = "3"
	switchResources := test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID1, switchID2, switchID3)
	publishedSwitches := test.ResourceLinksToResources(deviceID, switchResources)
	ev1 := <-dataChan
	links1 := ev1.GetData().(schema.ResourceLinks)
	ev2 := <-dataChan
	links2 := ev2.GetData().(schema.ResourceLinks)
	ev3 := <-dataChan
	links3 := ev3.GetData().(schema.ResourceLinks)
	resources = testSubscribeToDeviceDecodeResources(t, links1, links2, links3)
	test.CheckProtobufs(t, publishedSwitches, resources, test.RequireToCheckFunc(require.Equal))

	_, err = c.DeleteResource(ctx, &pb.DeleteResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID2)),
	})
	require.NoError(t, err)
	ev = <-dataChan
	links = ev.GetData().(schema.ResourceLinks)
	resources = testSubscribeToDeviceDecodeResources(t, links)
	var unpublishedSwitches []*commands.Resource
	for _, res := range resources {
		if res.Href == test.TestResourceSwitchesInstanceHref(switchID2) {
			unpublishedSwitches = append(unpublishedSwitches, &commands.Resource{
				DeviceId: deviceID,
				Href:     test.TestResourceSwitchesInstanceHref(switchID2),
			})
		}
	}
	test.CheckProtobufs(t, unpublishedSwitches, resources, test.RequireToCheckFunc(require.Equal))

	subscriber.Unsubscribe(t, ctx, token, deviceID, subID)
	ev = <-dataChan
	assert.Equal(t, events.EventType_SubscriptionCanceled, ev.GetHeader().EventType)
}
