package subscription_test

import (
	"context"
	"testing"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	subscription "github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
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
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
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
						test.LightResourceRepresentation{
							Name:  "Light",
							Power: value,
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
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, retrieveCorrelationID, oauthService.DeviceUserID),
			},
		},
		CorrelationId: correlationID,
	})
	check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_ResourceRetrieved{
			ResourceRetrieved: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceLightInstanceHref("1"),
				retrieveCorrelationID,
				test.LightResourceRepresentation{
					Name: "Light",
				},
			),
		},
		CorrelationId: correlationID,
	})
}

func getResourceChangedEvents(t *testing.T, deviceID, correlationID, subscriptionID string) map[string]*pb.Event {
	resources := test.GetAllBackendResourceRepresentations(t, deviceID, test.TestDeviceName)
	events := make(map[string]*pb.Event)
	for _, res := range resources {
		rid := commands.ResourceIdFromString(res.Href) // validate
		events[rid.GetHref()] = &pb.Event{
			SubscriptionId: subscriptionID,
			Type: &pb.Event_ResourceChanged{
				ResourceChanged: pbTest.MakeResourceChanged(t, deviceID, rid.GetHref(), "", res.Representation),
			},
			CorrelationId: correlationID,
		}
	}
	return events
}

func TestRequestHandlerSubscribeToEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithIncomingToken(kitNetGrpc.CtxWithToken(ctx, token), token)

	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	rdConn, err := grpcClient.New(ctx, config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = rdConn.Close()
	}()
	rdc := pb.NewGrpcGatewayClient(rdConn.GRPC())

	raConn, err := grpcClient.New(ctx, config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST), fileWatcher, log.Get(), noop.NewTracerProvider())
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

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, rdc, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)

	check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_DeviceRegistered_{
			DeviceRegistered: &pb.Event_DeviceRegistered{
				DeviceIds: []string{deviceID},
				EventMetadata: &isEvents.EventMetadata{
					HubId: config.HubID(),
				},
			},
		},
		CorrelationId: correlationID,
	})
	online := &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: pbTest.MakeDeviceMetadataUpdated(deviceID, commands.Connection_ONLINE, test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME), true, commands.TwinSynchronization_OUT_OF_SYNC, ""),
		},
		CorrelationId: correlationID,
	}
	onlineStartSync := &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: pbTest.MakeDeviceMetadataUpdated(deviceID, commands.Connection_ONLINE, test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME), true, commands.TwinSynchronization_SYNCING, ""),
		},
		CorrelationId: correlationID,
	}
	onlineFinishedSync := &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: pbTest.MakeDeviceMetadataUpdated(deviceID, commands.Connection_ONLINE, test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME), true, commands.TwinSynchronization_IN_SYNC, ""),
		},
		CorrelationId: correlationID,
	}
	publishedEv := pbTest.ResourceLinkToPublishEvent(deviceID, correlationID, test.GetAllBackendResourceLinks())
	expEvents := map[string]*pb.Event{
		pbTest.GetEventID(online):             online,
		pbTest.GetEventID(onlineStartSync):    onlineStartSync,
		pbTest.GetEventID(onlineFinishedSync): onlineFinishedSync,
		pbTest.GetEventID(publishedEv):        publishedEv,
	}
	resChanged := getResourceChangedEvents(t, deviceID, correlationID, s.Id())
	for _, v := range resChanged {
		expEvents[pbTest.GetEventID(v)] = v
	}
LOOP:
	for {
		select {
		case <-ctx.Done():
			require.Fail(t, "timeout")
		case e := <-recvChan:
			exp, ok := expEvents[pbTest.GetEventID(e)]
			if !ok {
				require.Failf(t, "unexpected event", "%v", e)
			}
			check(t, e, exp)
			delete(expEvents, pbTest.GetEventID(e))
			if len(expEvents) == 0 {
				break LOOP
			}
		}
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
						Connection: &commands.Connection{
							Status:   commands.Connection_OFFLINE,
							Protocol: test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
						},
						TwinSynchronization: &commands.TwinSynchronization{},
						TwinEnabled:         true,
						AuditContext:        commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
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
						EventMetadata: &isEvents.EventMetadata{
							HubId: config.HubID(),
						},
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
