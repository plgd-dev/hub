package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raservice "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

type testDeviceResourcesBaseObservationHandler struct {
	res chan interface{}
}

func makeTestDeviceResourcesBaseObservationHandler() testDeviceResourcesBaseObservationHandler {
	return testDeviceResourcesBaseObservationHandler{res: make(chan interface{}, 100)}
}

func (h *testDeviceResourcesBaseObservationHandler) Error(err error) {
	fmt.Println(err)
}

func (h *testDeviceResourcesBaseObservationHandler) OnClose() {
	fmt.Println("devices observation was closed")
}

func (h *testDeviceResourcesBaseObservationHandler) waitForEvents(t *testing.T, subID, correlationID string, eventsCount int) []*pb.Event {
	gotEvents := make([]*pb.Event, 0, eventsCount)
	for range eventsCount {
		select {
		case res := <-h.res:
			event := pbTest.ToEvent(res)
			require.NotNil(t, event)
			event.SubscriptionId = subID
			event.CorrelationId = correlationID
			gotEvents = append(gotEvents, event)
		case <-time.After(time.Second * 10):
			t.Error("timeout")
			return nil
		}
	}
	return gotEvents
}

func TestObserveDeviceResourcesRetrieve(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))
	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	raConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	rac := raservice.NewResourceAggregateClient(raConn.GRPC())

	c := grpcgwTest.NewTestClient(t)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	h := newTestDeviceResourcesRetrieveObservationHandler()
	sub, err := c.NewDeviceSubscription(ctx, deviceID, h)
	require.NoError(t, err)
	defer func() {
		wait, errC := sub.Cancel()
		require.NoError(t, errC)
		wait()
	}()

	const retrieveCorrelationID = "retrieveCorrelationID"
	_, err = rac.RetrieveResource(ctx, &commands.RetrieveResourceRequest{
		ResourceId:    commands.NewResourceID(deviceID, platform.ResourceURI),
		CorrelationId: retrieveCorrelationID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: "test",
		},
	})
	require.NoError(t, err)

	expectedEvents := []*pb.Event{
		{
			SubscriptionId: sub.ID(),
			CorrelationId:  retrieveCorrelationID,
			Type: &pb.Event_ResourceRetrievePending{
				ResourceRetrievePending: pbTest.MakeResourceRetrievePending(deviceID, platform.ResourceURI, []string{platform.ResourceType}, retrieveCorrelationID),
			},
		},
		{
			SubscriptionId: sub.ID(),
			CorrelationId:  retrieveCorrelationID,
			Type: &pb.Event_ResourceRetrieved{
				ResourceRetrieved: pbTest.MakeResourceRetrieved(t, deviceID, platform.ResourceURI, []string{platform.ResourceType}, retrieveCorrelationID,
					map[string]interface{}{
						"mnmn":                   "ocfcloud.com",
						"x.org.iotivity.version": test.GetIotivityLiteVersion(t, deviceID),
					}),
			},
		},
	}
	gotEvents := h.waitForEvents(t, sub.ID(), retrieveCorrelationID, len(expectedEvents))
	require.Len(t, gotEvents, len(expectedEvents))
	for i := range gotEvents {
		pbTest.CmpEvent(t, expectedEvents[i], gotEvents[i], "")
	}
}

type testDeviceResourcesRetrieveObservationHandler struct {
	testDeviceResourcesBaseObservationHandler
}

func newTestDeviceResourcesRetrieveObservationHandler() *testDeviceResourcesRetrieveObservationHandler {
	return &testDeviceResourcesRetrieveObservationHandler{
		testDeviceResourcesBaseObservationHandler: makeTestDeviceResourcesBaseObservationHandler(),
	}
}

func (h *testDeviceResourcesRetrieveObservationHandler) HandleResourceRetrievePending(_ context.Context, val *events.ResourceRetrievePending) error {
	h.res <- val
	return nil
}

func (h *testDeviceResourcesRetrieveObservationHandler) HandleResourceRetrieved(_ context.Context, val *events.ResourceRetrieved) error {
	h.res <- val
	return nil
}

func TestObserveDeviceResourcesUpdate(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))
	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	raConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	rac := raservice.NewResourceAggregateClient(raConn.GRPC())

	c := grpcgwTest.NewTestClient(t)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
	const switchID = "1"
	test.AddDeviceSwitchResources(ctx, t, deviceID, c.GrpcGatewayClient(), switchID)
	time.Sleep(time.Millisecond * 200)

	h := newTestDeviceResourcesUpdateObservationHandler()
	sub, err := c.NewDeviceSubscription(ctx, deviceID, h)
	require.NoError(t, err)
	defer func() {
		wait, errC := sub.Cancel()
		require.NoError(t, errC)
		wait()
	}()

	const updCorrelationID = "updCorrelationID"
	_, err = rac.UpdateResource(ctx, &commands.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID)),
		Content: &commands.Content{
			CoapContentFormat: -1,
			ContentType:       message.AppOcfCbor.String(),
			Data: func() []byte {
				v := map[string]interface{}{
					"value": true,
				}
				d, errEnc := cbor.Encode(v)
				require.NoError(t, errEnc)
				return d
			}(),
		},
		CorrelationId: updCorrelationID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: "test",
		},
	})
	require.NoError(t, err)

	expectedEvents := []*pb.Event{
		{
			SubscriptionId: sub.ID(),
			CorrelationId:  updCorrelationID,
			Type: &pb.Event_ResourceUpdatePending{
				ResourceUpdatePending: pbTest.MakeResourceUpdatePending(t, deviceID, test.TestResourceSwitchesInstanceHref(switchID), test.TestResourceSwitchesInstanceResourceTypes, updCorrelationID,
					map[string]interface{}{
						"value": true,
					}),
			},
		},
		{
			SubscriptionId: sub.ID(),
			CorrelationId:  updCorrelationID,
			Type: &pb.Event_ResourceUpdated{
				ResourceUpdated: &events.ResourceUpdated{
					ResourceId: &commands.ResourceId{
						DeviceId: deviceID,
						Href:     test.TestResourceSwitchesInstanceHref(switchID),
					},
					Status: commands.Status_OK,
					Content: &commands.Content{
						ContentType:       message.AppOcfCbor.String(),
						CoapContentFormat: int32(message.AppOcfCbor),
						Data: func() []byte {
							v := map[string]interface{}{
								"value": true,
							}
							d, err := cbor.Encode(v)
							require.NoError(t, err)
							return d
						}(),
					},
					AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, updCorrelationID, oauthService.DeviceUserID),
					ResourceTypes: test.TestResourceSwitchesInstanceResourceTypes,
				},
			},
		},
	}
	gotEvents := h.waitForEvents(t, sub.ID(), updCorrelationID, len(expectedEvents))
	require.Len(t, gotEvents, len(expectedEvents))
	for i := range gotEvents {
		pbTest.CmpEvent(t, expectedEvents[i], gotEvents[i], "")
	}
}

type testDeviceResourcesUpdateObservationHandler struct {
	testDeviceResourcesBaseObservationHandler
}

func newTestDeviceResourcesUpdateObservationHandler() *testDeviceResourcesUpdateObservationHandler {
	return &testDeviceResourcesUpdateObservationHandler{
		testDeviceResourcesBaseObservationHandler: makeTestDeviceResourcesBaseObservationHandler(),
	}
}

func (h *testDeviceResourcesUpdateObservationHandler) HandleResourceUpdatePending(_ context.Context, val *events.ResourceUpdatePending) error {
	h.res <- val
	return nil
}

func (h *testDeviceResourcesUpdateObservationHandler) HandleResourceUpdated(_ context.Context, val *events.ResourceUpdated) error {
	h.res <- val
	return nil
}

func TestObserveDeviceResourcesCreateAndDelete(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT*5)
	defer cancel()
	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))
	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	raConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	rac := raservice.NewResourceAggregateClient(raConn.GRPC())

	c := grpcgwTest.NewTestClient(t)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	h := newTestDeviceResourcesCreateAndDeleteObservationHandler()
	sub, err := c.NewDeviceSubscription(ctx, deviceID, h)
	require.NoError(t, err)
	defer func() {
		wait, errC := sub.Cancel()
		require.NoError(t, errC)
		wait()
	}()

	const switchID = "1"
	const createCorrelationID = "createCorrelationID"
	const connectionID = "testID"
	_, err = rac.CreateResource(ctx, &commands.CreateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesHref),
		Content: &commands.Content{
			CoapContentFormat: -1,
			ContentType:       message.AppOcfCbor.String(),
			Data:              test.EncodeToCbor(t, test.MakeSwitchResourceDefaultData()),
		},
		CorrelationId: createCorrelationID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
	})
	require.NoError(t, err)

	expectedCreateEvents := []*pb.Event{
		{
			SubscriptionId: sub.ID(),
			CorrelationId:  createCorrelationID,
			Type: &pb.Event_ResourceCreatePending{
				ResourceCreatePending: pbTest.MakeResourceCreatePending(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, createCorrelationID,
					test.MakeSwitchResourceDefaultData()),
			},
		},
		{
			SubscriptionId: sub.ID(),
			CorrelationId:  createCorrelationID,
			Type: &pb.Event_ResourceCreated{
				ResourceCreated: pbTest.MakeResourceCreated(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, createCorrelationID,
					pbTest.MakeCreateSwitchResourceResponseData(switchID)),
			},
		},
	}
	gotEvents := h.waitForEvents(t, sub.ID(), createCorrelationID, len(expectedCreateEvents))
	require.Len(t, gotEvents, len(expectedCreateEvents))
	for i := range gotEvents {
		pbTest.CmpEvent(t, expectedCreateEvents[i], gotEvents[i], "")
	}

	// give some time to the new resource to be initialized and all notifications to be handled
	// before running delete
	time.Sleep(time.Second * 2)

	const delCorrelationID = "delCorrelationID"
	_, err = rac.DeleteResource(ctx, &commands.DeleteResourceRequest{
		ResourceId:    commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID)),
		CorrelationId: delCorrelationID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
	})
	require.NoError(t, err)

	expectedDeleteEvents := []*pb.Event{
		{
			SubscriptionId: sub.ID(),
			CorrelationId:  delCorrelationID,
			Type: &pb.Event_ResourceDeletePending{
				ResourceDeletePending: pbTest.MakeResourceDeletePending(deviceID, test.TestResourceSwitchesInstanceHref(switchID), test.TestResourceSwitchesInstanceResourceTypes,
					delCorrelationID),
			},
		},
		{
			SubscriptionId: sub.ID(),
			CorrelationId:  delCorrelationID,
			Type: &pb.Event_ResourceDeleted{
				ResourceDeleted: pbTest.MakeResourceDeleted(deviceID, test.TestResourceSwitchesInstanceHref(switchID), test.TestResourceSwitchesInstanceResourceTypes,
					delCorrelationID),
			},
		},
	}
	gotEvents = h.waitForEvents(t, sub.ID(), delCorrelationID, len(expectedDeleteEvents))
	require.Len(t, gotEvents, len(expectedDeleteEvents))
	for i := range gotEvents {
		pbTest.CmpEvent(t, expectedDeleteEvents[i], gotEvents[i], "")
	}
}

type testDeviceResourcesCreateAndDeleteObservationHandler struct {
	testDeviceResourcesBaseObservationHandler
}

func newTestDeviceResourcesCreateAndDeleteObservationHandler() *testDeviceResourcesCreateAndDeleteObservationHandler {
	return &testDeviceResourcesCreateAndDeleteObservationHandler{
		testDeviceResourcesBaseObservationHandler: makeTestDeviceResourcesBaseObservationHandler(),
	}
}

func (h *testDeviceResourcesCreateAndDeleteObservationHandler) HandleResourceCreatePending(_ context.Context, val *events.ResourceCreatePending) error {
	h.res <- val
	return nil
}

func (h *testDeviceResourcesCreateAndDeleteObservationHandler) HandleResourceCreated(_ context.Context, val *events.ResourceCreated) error {
	h.res <- val
	return nil
}

func (h *testDeviceResourcesCreateAndDeleteObservationHandler) HandleResourceDeletePending(_ context.Context, val *events.ResourceDeletePending) error {
	h.res <- val
	return nil
}

func (h *testDeviceResourcesCreateAndDeleteObservationHandler) HandleResourceDeleted(_ context.Context, val *events.ResourceDeleted) error {
	h.res <- val
	return nil
}
