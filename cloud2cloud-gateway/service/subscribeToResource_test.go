package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	c2cEvents "github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	c2cTest "github.com/plgd-dev/hub/v2/cloud2cloud-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerSubscribeToResource(t *testing.T) {
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

	subscriber := c2cTest.NewC2CSubscriber(eventsServer.GetPort(t), eventsURI, c2cTest.SubscriptionType_Resource)
	subID := subscriber.Subscribe(ctx, t, token, deviceID, test.TestResourceLightInstanceHref("1"),
		c2cEvents.EventTypes{c2cEvents.EventType_ResourceChanged})

	events := c2cTest.WaitForEvents(dataChan, 3*time.Second)
	require.Len(t, events, 1)
	assert.Equal(t, c2cEvents.EventType_ResourceChanged, events[0].GetHeader().EventType)
	wantEventContent := map[interface{}]interface{}{
		"name":  "Light",
		"power": uint64(0),
		"state": false,
	}
	assert.Equal(t, wantEventContent, events[0].GetData())

	subscriber.Unsubscribe(ctx, t, token, deviceID, test.TestResourceLightInstanceHref("1"), subID)
	ev := <-dataChan
	assert.Equal(t, c2cEvents.EventType_SubscriptionCanceled, ev.GetHeader().EventType)
}

func TestRequestHandlerSubscribeToResourceTokenTimeout(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	services := service.SetUpServicesOAuth | service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesId | service.SetUpServicesCertificateAuthority |
		service.SetUpServicesResourceAggregate | service.SetUpServicesResourceDirectory | service.SetUpServicesGrpcGateway |
		service.SetUpServicesCoapGateway
	tearDown := service.SetUpServices(ctx, t, services)
	defer tearDown()
	c2cgwShutdown := c2cTest.SetUp(t)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()

	token := oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTestShortExpiration, nil)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	_, shutdownDevSim := test.OnboardDevSimForClient(ctx, t, c, oauthTest.ClientTestShortExpiration, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST,
		test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	const eventsURI = "/events"
	eventsServer := c2cTest.NewEventsServer(t, eventsURI)
	defer eventsServer.Close(t)
	dataChan := eventsServer.Run(t)

	subscriber := c2cTest.NewC2CSubscriber(eventsServer.GetPort(t), eventsURI, c2cTest.SubscriptionType_Resource)
	_ = subscriber.Subscribe(ctx, t, token, deviceID, test.TestResourceLightInstanceHref("1"),
		c2cEvents.EventTypes{c2cEvents.EventType_ResourceChanged})

	events := c2cTest.WaitForEvents(dataChan, 3*time.Second)
	require.Len(t, events, 1)
	assert.Equal(t, c2cEvents.EventType_ResourceChanged, events[0].GetHeader().EventType)

	// let access token expire
	time.Sleep(time.Second * 10)
	// stop and start c2c-gw and let it try reestablish resource subscription with expired token
	c2cgwShutdown()
	time.Sleep(time.Second)
	c2cgwShutdown = c2cTest.SetUp(t)
	defer c2cgwShutdown()

	events = c2cTest.WaitForEvents(dataChan, 6*time.Second)
	require.Len(t, events, 1)
	assert.Equal(t, c2cEvents.EventType_SubscriptionCanceled, events[0].GetHeader().EventType)
}
