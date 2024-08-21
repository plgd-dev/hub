package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func subscribeToAllEvents(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, correlationID string) (pb.GrpcGateway_SubscribeToEventsClient, string) {
	subClient, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)
	err = subClient.Send(&pb.SubscribeToEvents{
		CorrelationId: correlationID,
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{},
		},
	})
	require.NoError(t, err)
	ev, err := subClient.Recv()
	require.NoError(t, err)
	test.CheckProtobufs(t, pbTest.NewOperationProcessedOK(ev.GetSubscriptionId(), correlationID), ev, test.RequireToCheckFunc(require.Equal))
	return subClient, ev.GetSubscriptionId()
}

func createSwitchResource(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, deviceID, switchID string) {
	got, err := c.CreateResource(ctx, &pb.CreateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesHref),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data:        test.EncodeToCbor(t, test.MakeSwitchResourceDefaultData()),
		},
	})
	require.NoError(t, err)
	switchData := pbTest.MakeCreateSwitchResourceResponseData(switchID)
	want := pbTest.MakeResourceCreated(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, "", switchData)
	pbTest.CmpResourceCreated(t, want, got.GetData())
}

func deleteSwitchResource(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, deviceID, switchID string) {
	got, err := c.DeleteResource(ctx, &pb.DeleteResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID)),
	})
	require.NoError(t, err)
	want := pbTest.MakeResourceDeleted(deviceID, test.TestResourceSwitchesInstanceHref(switchID), test.TestResourceSwitchesInstanceResourceTypes, "")
	pbTest.CmpResourceDeleted(t, want, got.GetData())
}

func createSwitchResourceExpectedEvents(t *testing.T, deviceID, subID, correlationID, switchID string, isDiscoveryResourceBatchObservable bool) map[string]*pb.Event {
	cpEvent := &pb.Event{
		SubscriptionId: subID,
		CorrelationId:  correlationID,
		Type: &pb.Event_ResourceCreatePending{
			ResourceCreatePending: pbTest.MakeResourceCreatePending(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, "",
				test.MakeSwitchResourceDefaultData()),
		},
	}

	rchangedEvent := &pb.Event{
		SubscriptionId: subID,
		CorrelationId:  correlationID,
		Type: &pb.Event_ResourceChanged{
			ResourceChanged: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, "",
				[]map[string]interface{}{
					{
						"href": test.TestResourceSwitchesInstanceHref(switchID),
						"if":   []string{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
						"p": map[interface{}]interface{}{
							"bm": uint64(schema.Discoverable | schema.Observable),
						},
						"rel": []string{"hosts"},
						"rt":  []string{types.BINARY_SWITCH},
					},
				},
			),
		},
	}

	rcreatedEvent := &pb.Event{
		SubscriptionId: subID,
		CorrelationId:  correlationID,
		Type: &pb.Event_ResourceCreated{
			ResourceCreated: pbTest.MakeResourceCreated(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, "",
				test.MakeSwitchResourceData(map[string]interface{}{
					"href": test.TestResourceSwitchesInstanceHref(switchID),
					"rep": map[string]interface{}{
						"if":    []string{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
						"rt":    []string{types.BINARY_SWITCH},
						"value": false,
					},
				}),
			),
		},
	}

	rpublishedEvent := &pb.Event{
		SubscriptionId: subID,
		CorrelationId:  correlationID,
		Type: &pb.Event_ResourcePublished{
			ResourcePublished: &events.ResourceLinksPublished{
				DeviceId: deviceID,
				Resources: []*commands.Resource{
					{
						Href:          test.TestResourceSwitchesInstanceHref(switchID),
						DeviceId:      deviceID,
						ResourceTypes: []string{types.BINARY_SWITCH},
						Interfaces:    []string{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
						Policy: &commands.Policy{
							BitFlags: commands.ToPolicyBitFlags(schema.Discoverable | schema.Observable),
						},
					},
				},
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
			},
		},
	}

	rchangedEvent2 := &pb.Event{
		SubscriptionId: subID,
		CorrelationId:  correlationID,
		Type: &pb.Event_ResourceChanged{
			ResourceChanged: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesInstanceHref(switchID), test.TestResourceSwitchesInstanceResourceTypes, "",
				map[string]interface{}{
					"value": false,
				}),
		},
	}

	ret := map[string]*pb.Event{
		pbTest.GetEventID(cpEvent):         cpEvent,
		pbTest.GetEventID(rchangedEvent):   rchangedEvent,
		pbTest.GetEventID(rcreatedEvent):   rcreatedEvent,
		pbTest.GetEventID(rpublishedEvent): rpublishedEvent,
		pbTest.GetEventID(rchangedEvent2):  rchangedEvent2,
	}

	if !isDiscoveryResourceBatchObservable {
		rsSyncStartedEvent := &pb.Event{
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_DeviceMetadataUpdated{
				DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
					DeviceId: deviceID,
					Connection: &commands.Connection{
						Status:   commands.Connection_ONLINE,
						Protocol: test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
					},
					TwinEnabled: true,
					TwinSynchronization: &commands.TwinSynchronization{
						State: commands.TwinSynchronization_SYNCING,
					},
					AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
				},
			},
		}

		rsSyncFinishedEvent := &pb.Event{
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_DeviceMetadataUpdated{
				DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
					DeviceId: deviceID,
					Connection: &commands.Connection{
						Status:   commands.Connection_ONLINE,
						Protocol: test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
					},
					TwinEnabled: true,
					TwinSynchronization: &commands.TwinSynchronization{
						State: commands.TwinSynchronization_IN_SYNC,
					},
					AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
				},
			},
		}
		ret[pbTest.GetEventID(rsSyncStartedEvent)] = rsSyncStartedEvent
		ret[pbTest.GetEventID(rsSyncFinishedEvent)] = rsSyncFinishedEvent
	}
	return ret
}

func deleteSwitchResourceExpectedEvents(t *testing.T, deviceID, subID, correlationID, switchID string) map[string]*pb.Event {
	deletePending := &pb.Event{
		SubscriptionId: subID,
		CorrelationId:  correlationID,
		Type: &pb.Event_ResourceDeletePending{
			ResourceDeletePending: &events.ResourceDeletePending{
				ResourceId:    commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID)),
				AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
				ResourceTypes: test.TestResourceSwitchesInstanceResourceTypes,
			},
		},
	}

	deleted := &pb.Event{
		SubscriptionId: subID,
		CorrelationId:  correlationID,
		Type: &pb.Event_ResourceDeleted{
			ResourceDeleted: pbTest.MakeResourceDeleted(deviceID, test.TestResourceSwitchesInstanceHref(switchID), test.TestResourceSwitchesInstanceResourceTypes, ""),
		},
	}

	unpublished := &pb.Event{
		SubscriptionId: subID,
		CorrelationId:  correlationID,
		Type: &pb.Event_ResourceUnpublished{
			ResourceUnpublished: &events.ResourceLinksUnpublished{
				DeviceId:     deviceID,
				Hrefs:        []string{test.TestResourceSwitchesInstanceHref(switchID)},
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
			},
		},
	}

	changed := &pb.Event{
		SubscriptionId: subID,
		CorrelationId:  correlationID,
		Type: &pb.Event_ResourceChanged{
			ResourceChanged: pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, "", []interface{}{}),
		},
	}

	res := pbTest.MakeResourceChanged(t, deviceID, test.TestResourceSwitchesInstanceHref(switchID), test.TestResourceSwitchesInstanceResourceTypes, "", nil)
	res.Status = commands.Status_NOT_FOUND
	res.Content.CoapContentFormat = -1
	res.Content.ContentType = ""
	changedRes := &pb.Event{
		SubscriptionId: subID,
		CorrelationId:  correlationID,
		Type: &pb.Event_ResourceChanged{
			ResourceChanged: res,
		},
	}

	e := map[string]*pb.Event{
		pbTest.GetEventID(deletePending): deletePending,
		pbTest.GetEventID(deleted):       deleted,
		pbTest.GetEventID(unpublished):   unpublished,
		pbTest.GetEventID(changed):       changed,
		pbTest.GetEventID(changedRes):    changedRes,
	}

	return e
}

func validateEvents(t *testing.T, subClient pb.GrpcGateway_SubscribeToEventsClient, expectedEvents map[string]*pb.Event) {
	for {
		ev, err := subClient.Recv()
		if kitNetGrpc.IsContextDeadlineExceeded(err) {
			require.Failf(t, "missing events", "expected events not received: %+v", expectedEvents)
		}
		require.NoError(t, err)

		eventID := pbTest.GetEventID(ev)
		expected, ok := expectedEvents[eventID]
		if !ok {
			require.Failf(t, "unexpected event", "invalid event: %+v", ev)
		}
		pbTest.CmpEvent(t, expected, ev, "")
		delete(expectedEvents, eventID)
		if len(expectedEvents) == 0 {
			break
		}
	}
}

func TestCreateAndDeleteResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := hubTestService.SetUp(ctx, t)
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

	const correlationID = "allEvents"
	subClient, subID := subscribeToAllEvents(ctx, t, c, correlationID)
	const switchID = "1"

	isDiscoveryResourceBatchObservable := test.IsDiscoveryResourceBatchObservable(ctx, t, deviceID)
	for i := 0; i < 5; i++ {
		fmt.Printf("iteration %v\n", i)
		// for update resource-directory cache
		time.Sleep(time.Second)
		createSwitchResource(ctx, t, c, deviceID, switchID)
		expectedCreateEvents := createSwitchResourceExpectedEvents(t, deviceID, subID, correlationID, switchID, isDiscoveryResourceBatchObservable)
		validateEvents(t, subClient, expectedCreateEvents)
		deleteSwitchResource(ctx, t, c, deviceID, switchID)
		expectedDeleteEvents := deleteSwitchResourceExpectedEvents(t, deviceID, subID, correlationID, switchID)
		validateEvents(t, subClient, expectedDeleteEvents)
	}
}
