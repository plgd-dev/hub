package subscription_test

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/softwareupdate"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	subscription "github.com/plgd-dev/hub/v2/grpc-gateway/subscription"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raservice "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
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
		expectedEvent.SubscriptionId = ev.GetSubscriptionId()
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

	resourceUpdatePending := pbTest.MakeResourceUpdatePending(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, updCorrelationID,
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
	for range 2 {
		ev := waitForEvent(ctx, t, recvChan)
		switch {
		case ev.GetResourceUpdated() != nil:
			check(t, ev, &pb.Event{
				SubscriptionId: s.Id(),
				Type: &pb.Event_ResourceUpdated{
					ResourceUpdated: pbTest.MakeResourceUpdated(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, updCorrelationID, nil),
				},
				CorrelationId: correlationID,
			})
		case ev.GetResourceChanged() != nil:
			check(t, ev, &pb.Event{
				SubscriptionId: s.Id(),
				Type: &pb.Event_ResourceChanged{
					ResourceChanged: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "",
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
				ResourceId:    commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
				AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, retrieveCorrelationID, oauthService.DeviceUserID),
				ResourceTypes: test.TestResourceLightInstanceResourceTypes,
			},
		},
		CorrelationId: correlationID,
	})
	check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_ResourceRetrieved{
			ResourceRetrieved: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes,
				retrieveCorrelationID,
				test.LightResourceRepresentation{
					Name: "Light",
				},
			),
		},
		CorrelationId: correlationID,
	})
}

func getResourceChangedEvents(t *testing.T, deviceID, correlationID, subscriptionID string, filter func(href string) bool) map[string]*pb.Event {
	resources := test.GetAllBackendResourceRepresentations(t, deviceID, test.TestDeviceName)
	events := make(map[string]*pb.Event)
	for _, res := range resources {
		rid := commands.ResourceIdFromString(res.Href) // validate
		if filter != nil && !filter(rid.GetHref()) {
			continue
		}
		events[rid.GetHref()] = &pb.Event{
			SubscriptionId: subscriptionID,
			Type: &pb.Event_ResourceChanged{
				ResourceChanged: pbTest.MakeResourceChanged(t, deviceID, rid.GetHref(), res.ResourceTypes, "", res.Representation),
			},
			CorrelationId: correlationID,
		}
	}
	return events
}

func prepareServicesAndSubscription(t *testing.T, owner, correlationID string, leadRTEnabled bool, req *pb.SubscribeToEvents_CreateSubscription, sendEvent subscription.SendEventFunc) (pb.GrpcGatewayClient, raservice.ResourceAggregateClient, *subscription.Sub, func()) {
	var cleanUp fn.FuncList
	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	cleanUp.AddFunc(func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	})

	rdConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	cleanUp.AddFunc(func() {
		_ = rdConn.Close()
	})
	rdc := pb.NewGrpcGatewayClient(rdConn.GRPC())

	raConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	cleanUp.AddFunc(func() {
		_ = raConn.Close()
	})
	rac := raservice.NewResourceAggregateClient(raConn.GRPC())

	pool, err := ants.NewPool(1)
	require.NoError(t, err)
	natsConn, resourceSubscriber, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), fileWatcher, log.Get(), noop.NewTracerProvider(), subscriber.WithGoPool(pool.Submit), subscriber.WithUnmarshaler(utils.Unmarshal))
	require.NoError(t, err)
	cleanUp.AddFunc(func() {
		resourceSubscriber.Close()
		natsConn.Close()
	})

	subCache := subscription.NewSubscriptionsCache(resourceSubscriber.Conn(), func(err error) { log.Get().Error(err) })

	s := subscription.New(sendEvent, correlationID, leadRTEnabled, req)
	err = s.Init(owner, subCache)
	require.NoError(t, err)
	cleanUp.AddFunc(func() {
		errC := s.Close()
		require.NoError(t, errC)
	})

	return rdc, rac, s, cleanUp.ToFunction()
}

func TestRequestHandlerSubscribeToEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithIncomingToken(kitNetGrpc.CtxWithToken(ctx, token), token)

	owner, err := kitNetGrpc.OwnerFromTokenMD(ctx, config.OWNER_CLAIM)
	require.NoError(t, err)

	correlationID := "testToken"
	recvChan := make(chan *pb.Event, 1)
	rdc, rac, s, cleanUp := prepareServicesAndSubscription(t, owner, correlationID, false, &pb.SubscribeToEvents_CreateSubscription{},
		func(e *pb.Event) error {
			select {
			case recvChan <- e:
			case <-ctx.Done():
			}
			return nil
		})
	defer cleanUp()

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
	resChanged := getResourceChangedEvents(t, deviceID, correlationID, s.Id(), nil)
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

func testRequestHandlerSubscribeToChangedEvents(t *testing.T, cfg *natsClient.LeadResourceTypePublisherConfig, createSubscription *pb.SubscribeToEvents_CreateSubscription, filterExpectedEvents func(href string) bool) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	raCfg := raTest.MakeConfig(t)
	raCfg.Clients.Eventbus.NATS.LeadResourceType = cfg
	err := raCfg.Clients.Eventbus.NATS.Validate()
	require.NoError(t, err)
	tearDown := service.SetUp(ctx, t, service.WithRAConfig(raCfg))
	defer tearDown()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithIncomingToken(kitNetGrpc.CtxWithToken(ctx, token), token)

	owner, err := kitNetGrpc.OwnerFromTokenMD(ctx, config.OWNER_CLAIM)
	require.NoError(t, err)

	correlationID := uuid.NewString()
	recvChan := make(chan *pb.Event, 1)
	rdc, _, s, cleanUp := prepareServicesAndSubscription(t, owner, correlationID, true, createSubscription,
		func(e *pb.Event) error {
			select {
			case recvChan <- e:
			case <-ctx.Done():
			}
			return nil
		})
	defer cleanUp()

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, rdc, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	defer shutdownDevSim()

	expEvents := make(map[string]*pb.Event)
	resChanged := getResourceChangedEvents(t, deviceID, correlationID, s.Id(), filterExpectedEvents)
	for _, v := range resChanged {
		expEvents[pbTest.GetEventID(v)] = v
	}

	run := true
	for run {
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
				run = false
				continue
			}
		}
	}
}

func TestRequestHandlerSubscribeToAllResourceEvents(t *testing.T) {
	cfg := &natsClient.LeadResourceTypePublisherConfig{
		Enabled: true,
		Filter:  natsClient.LeadResourceTypeFilter_First,
	}
	createSub := &pb.SubscribeToEvents_CreateSubscription{
		EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
			pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING,
			pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATED,
			pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING,
			pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVED,
			pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETE_PENDING,
			pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETED,
			pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATE_PENDING,
			pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATED,
			pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
		},
		// NATS supports "*" and ">" wildcards, we must use ">" because a resource types can contain multiple substrings delimited by "."
		LeadResourceTypeFilter: []string{">"},
	}
	testRequestHandlerSubscribeToChangedEvents(t, cfg, createSub, nil)
}

func TestRequestHandlerSubscribeToResourceChangedEventsWithLeadResourceTypeLastAndUseUUID(t *testing.T) {
	cfg := &natsClient.LeadResourceTypePublisherConfig{
		Enabled: true,
		Filter:  natsClient.LeadResourceTypeFilter_Last,
		UseUUID: true,
	}
	lrtFilter := make([]string, 0, len(test.TestResourceLightInstanceResourceTypes))
	for _, rt := range test.TestResourceLightInstanceResourceTypes {
		lrtFilter = append(lrtFilter, publisher.ResourceTypeToUUID(rt))
	}
	createSub := &pb.SubscribeToEvents_CreateSubscription{
		EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
			pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
		},
		LeadResourceTypeFilter: lrtFilter,
	}
	testRequestHandlerSubscribeToChangedEvents(t, cfg, createSub, func(href string) bool {
		return test.TestResourceLightInstanceHref("1") == href
	})
}

func TestRequestHandlerSubscribeToResourceChangedEventsWithLeadResourceTypeRegex(t *testing.T) {
	// must match the regexFilter
	expectedHrefs := []string{test.TestResourceLightInstanceHref("1"), softwareupdate.ResourceURI, configuration.ResourceURI}
	cfg := &natsClient.LeadResourceTypePublisherConfig{
		Enabled:     true,
		RegexFilter: []string{"core\\.light", "oic\\.r.*", ".*\\.con", "x\\.plgd\\.dev\\..*"}, // x.plgd.dev.* is extra, it is published, but not subscribed to
	}
	createSub := &pb.SubscribeToEvents_CreateSubscription{
		EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
			pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
		},
		LeadResourceTypeFilter: []string{types.CORE_LIGHT, softwareupdate.ResourceType, configuration.ResourceType},
	}
	testRequestHandlerSubscribeToChangedEvents(t, cfg, createSub, func(href string) bool {
		return slices.Contains(expectedHrefs, href)
	})
}
