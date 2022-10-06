package subscription_test

import (
	"context"
	"testing"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/device/schema/configuration"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/platform"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	subscription "github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raservice "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func waitForEvent(ctx context.Context, t *testing.T, recvChan <-chan *pb.Event) *pb.Event {
	select {
	case ev := <-recvChan:
		pbTest.CleanUpEvent(t, ev)
		return ev
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}
	return nil
}

func check(t *testing.T, ev *pb.Event, expectedEvent *pb.Event) {
	if expectedEvent.GetResourcePublished() != nil {
		expectedEvent.SubscriptionId = ev.SubscriptionId
	}
	pbTest.CmpEvent(t, expectedEvent, ev, "")
}

func checkResourceChanged(t *testing.T, ev *pb.Event, expectedEvents map[string]*pb.Event) {
	if ev.GetResourceChanged() != nil {
		expectedEvent := expectedEvents[ev.GetResourceChanged().GetResourceId().GetHref()]
		pbTest.CmpEvent(t, expectedEvent, ev, "")
		return
	}
	assert.Fail(t, "unexpected event", "event: %v", ev)
}

func checkAndValidateUpdate(ctx context.Context, t *testing.T, rac raservice.ResourceAggregateClient, s *subscription.Sub, recvChan <-chan *pb.Event, correlationID, deviceID string, value uint64) {
	const updCorrelationID = "updCorrelationID"
	_, err := rac.UpdateResource(ctx, &commands.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &commands.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: func() []byte {
				v := map[string]interface{}{
					"power": value,
				}
				d, err := cbor.Encode(v)
				require.NoError(t, err)
				return d
			}(),
		},
		CorrelationId: updCorrelationID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: "test",
		},
	})
	require.NoError(t, err)

	resourceUpdatePending := pbTest.MakeResourceUpdatePending(t, deviceID, test.TestResourceLightInstanceHref("1"), updCorrelationID,
		map[string]interface{}{
			"power": value,
		},
	)
	resourceUpdatePending.GetContent().CoapContentFormat = 0
	check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_ResourceUpdatePending{
			ResourceUpdatePending: resourceUpdatePending,
		},
		CorrelationId: correlationID,
	})
	for i := 0; i < 2; i++ {
		ev := waitForEvent(ctx, t, recvChan)
		switch {
		case ev.GetResourceUpdated() != nil:
			check(t, ev, &pb.Event{
				SubscriptionId: s.Id(),
				Type: &pb.Event_ResourceUpdated{
					ResourceUpdated: pbTest.MakeResourceUpdated(t, deviceID, test.TestResourceLightInstanceHref("1"), updCorrelationID, nil),
				},
				CorrelationId: correlationID,
			})
		case ev.GetResourceChanged() != nil:
			check(t, ev, &pb.Event{
				SubscriptionId: s.Id(),
				Type: &pb.Event_ResourceChanged{
					ResourceChanged: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceLightInstanceHref("1"), "",
						map[string]interface{}{
							"name":  "Light",
							"power": value,
							"state": false,
						},
					),
				},
				CorrelationId: correlationID,
			})
		}
	}
}

func checkAndValidateRetrieve(ctx context.Context, t *testing.T, rac raservice.ResourceAggregateClient, s *subscription.Sub, recvChan <-chan *pb.Event, correlationID, deviceID string) {
	const retrieveCorrelationID = "retrieveCorrelationID"
	_, err := rac.RetrieveResource(ctx, &commands.RetrieveResourceRequest{
		ResourceId:    commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		CorrelationId: retrieveCorrelationID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: "test",
		},
	})
	require.NoError(t, err)

	check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_ResourceRetrievePending{
			ResourceRetrievePending: &events.ResourceRetrievePending{
				ResourceId:   commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, retrieveCorrelationID),
			},
		},
		CorrelationId: correlationID,
	})
	check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_ResourceRetrieved{
			ResourceRetrieved: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceLightInstanceHref("1"),
				retrieveCorrelationID,
				map[string]interface{}{
					"name":  "Light",
					"power": uint64(0),
					"state": false,
				},
			),
		},
		CorrelationId: correlationID,
	})
}

func getResourceChangedEvents(t *testing.T, deviceID, correlationID, subscriptionID string) map[string]*pb.Event {
	return map[string]*pb.Event{
		device.ResourceURI: {
			SubscriptionId: subscriptionID,
			Type: &pb.Event_ResourceChanged{
				ResourceChanged: pbTest.MakeResourceChanged(t, deviceID, device.ResourceURI, "", map[string]interface{}{
					"di":  deviceID,
					"dmv": "ocf.res.1.3.0",
					"icv": "ocf.2.0.5",
					"n":   test.TestDeviceName,
				}),
			},
			CorrelationId: correlationID,
		},
		platform.ResourceURI: {
			SubscriptionId: subscriptionID,
			Type: &pb.Event_ResourceChanged{
				ResourceChanged: pbTest.MakeResourceChanged(t, deviceID, platform.ResourceURI, "", map[string]interface{}{
					"mnmn": "ocfcloud.com",
				}),
			},
			CorrelationId: correlationID,
		},
		configuration.ResourceURI: {
			SubscriptionId: subscriptionID,
			Type: &pb.Event_ResourceChanged{
				ResourceChanged: pbTest.MakeResourceChanged(t, deviceID, configuration.ResourceURI, "", map[string]interface{}{
					"n": test.TestDeviceName,
				}),
			},
			CorrelationId: correlationID,
		},
		test.TestResourceLightInstanceHref("1"): {
			SubscriptionId: subscriptionID,
			Type: &pb.Event_ResourceChanged{
				ResourceChanged: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceLightInstanceHref("1"), "",
					map[string]interface{}{
						"name":  "Light",
						"power": uint64(0),
						"state": false,
					},
				),
			},
			CorrelationId: correlationID,
		},
		test.TestResourceSwitchesHref: {
			SubscriptionId: subscriptionID,
			Type: &pb.Event_ResourceChanged{
				ResourceChanged: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesHref, "",
					[]interface{}{},
				),
			},
			CorrelationId: correlationID,
		},
	}
}

func TestRequestHandlerSubscribeToEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithIncomingToken(kitNetGrpc.CtxWithToken(ctx, token), token)

	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	rdConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST), fileWatcher, log.Get(), trace.NewNoopTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = rdConn.Close()
	}()
	rdc := pb.NewGrpcGatewayClient(rdConn.GRPC())

	raConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST), fileWatcher, log.Get(), trace.NewNoopTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	rac := raservice.NewResourceAggregateClient(raConn.GRPC())

	pool, err := ants.NewPool(1)
	require.NoError(t, err)
	natsConn, resourceSubscriber, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), fileWatcher, log.Get(), subscriber.WithGoPool(pool.Submit), subscriber.WithUnmarshaler(utils.Unmarshal))
	require.NoError(t, err)
	defer natsConn.Close()
	defer resourceSubscriber.Close()

	owner, err := kitNetGrpc.OwnerFromTokenMD(ctx, config.OWNER_CLAIM)
	require.NoError(t, err)
	subCache := subscription.NewSubscriptionsCache(resourceSubscriber.Conn(), func(err error) { t.Log(err) })
	correlationID := "testToken"
	recvChan := make(chan *pb.Event, 1)

	s := subscription.New(func(e *pb.Event) error {
		select {
		case recvChan <- e:
		case <-ctx.Done():
		}
		return nil
	}, correlationID, &pb.SubscribeToEvents_CreateSubscription{})
	err = s.Init(owner, subCache)
	require.NoError(t, err)
	defer func() {
		err := s.Close()
		require.NoError(t, err)
	}()

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, rdc, deviceID, config.GW_HOST, nil)

	check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_DeviceRegistered_{
			DeviceRegistered: &pb.Event_DeviceRegistered{
				DeviceIds: []string{deviceID},
			},
		},
		CorrelationId: correlationID,
	})
	check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: pbTest.MakeDeviceMetadataUpdated(deviceID, commands.ShadowSynchronization_UNSET, ""),
		},
		CorrelationId: correlationID,
	})
	check(t, waitForEvent(ctx, t, recvChan), pbTest.ResourceLinkToPublishEvent(deviceID, correlationID, test.GetAllBackendResourceLinks()))

	expectedEvents := getResourceChangedEvents(t, deviceID, correlationID, s.Id())
	for range expectedEvents {
		checkResourceChanged(t, waitForEvent(ctx, t, recvChan), expectedEvents)
	}

	checkAndValidateUpdate(ctx, t, rac, s, recvChan, correlationID, deviceID, 99)
	checkAndValidateUpdate(ctx, t, rac, s, recvChan, correlationID, deviceID, 0)
	checkAndValidateRetrieve(ctx, t, rac, s, recvChan, correlationID, deviceID)

	shutdownDevSim()

	run := true
	for run {
		ev := waitForEvent(ctx, t, recvChan)
		switch {
		case ev.GetDeviceMetadataUpdated() != nil:
			check(t, ev, &pb.Event{
				SubscriptionId: s.Id(),
				Type: &pb.Event_DeviceMetadataUpdated{
					DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
						DeviceId: deviceID,
						Status: &commands.ConnectionStatus{
							Value: commands.ConnectionStatus_OFFLINE,
						},
						AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
					},
				},
				CorrelationId: correlationID,
			})
		case ev.GetDeviceUnregistered() != nil:
			check(t, ev, &pb.Event{
				SubscriptionId: s.Id(),
				Type: &pb.Event_DeviceUnregistered_{
					DeviceUnregistered: &pb.Event_DeviceUnregistered{
						DeviceIds: []string{deviceID},
					},
				},
				CorrelationId: correlationID,
			})
			run = false
		case ctx.Err() != nil:
			require.NoError(t, err)
		}
	}
}
