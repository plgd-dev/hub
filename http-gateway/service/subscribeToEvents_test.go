package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/go-coap/v3/message"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	rdTest "github.com/plgd-dev/hub/v2/resource-directory/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/sjson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func isDeviceMetadataUpdatedOnlineEvent(ev *pb.Event, deviceID string) bool {
	return ev.GetDeviceMetadataUpdated() != nil &&
		ev.GetDeviceMetadataUpdated().GetDeviceId() == deviceID &&
		ev.GetDeviceMetadataUpdated().GetConnection().GetStatus() == commands.Connection_ONLINE &&
		ev.GetDeviceMetadataUpdated().GetTwinSynchronization().GetState() == commands.TwinSynchronization_IN_SYNC
}

func checkDeviceMetadataUpdatedOnlineEvent(t *testing.T, ev *pb.Event, deviceID, baseSubID string, twinSynchronizationState commands.TwinSynchronization_State) {
	expectedEvent := &pb.Event{
		SubscriptionId: baseSubID,
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: pbTest.MakeDeviceMetadataUpdated(deviceID, commands.Connection_ONLINE, test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME), true, twinSynchronizationState, ""),
		},
		CorrelationId: "testToken",
	}
	pbTest.CmpEvent(t, expectedEvent, ev, "")
}

type updateChecker struct {
	c            pb.GrpcGatewayClient
	deviceID     string
	baseSubID    string
	subUpdatedID string

	recv func() (*pb.Event, error)
}

// update light resource and check received events
func (u *updateChecker) checkUpdateLightResource(ctx context.Context, t *testing.T, power uint64) {
	_, err := u.c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(u.deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": power,
			}),
		},
	})
	require.NoError(t, err)

	var updCorrelationID string
	for range 3 {
		ev, err := u.recv()
		require.NoError(t, err)
		switch {
		case ev.GetResourceUpdatePending() != nil:
			updCorrelationID = ev.GetResourceUpdatePending().GetAuditContext().GetCorrelationId()
			expectedEvent := &pb.Event{
				SubscriptionId: u.subUpdatedID,
				Type: &pb.Event_ResourceUpdatePending{
					ResourceUpdatePending: pbTest.MakeResourceUpdatePending(t, u.deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, updCorrelationID,
						map[string]interface{}{
							"power": power,
						}),
				},
				CorrelationId: "updatePending + resourceUpdated",
			}
			pbTest.CmpEvent(t, expectedEvent, ev, "")
		case ev.GetResourceUpdated() != nil:
			expectedEvent := &pb.Event{
				SubscriptionId: u.subUpdatedID,
				Type: &pb.Event_ResourceUpdated{
					ResourceUpdated: pbTest.MakeResourceUpdated(t, u.deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, updCorrelationID, nil),
				},
				CorrelationId: "updatePending + resourceUpdated",
			}
			pbTest.CmpEvent(t, expectedEvent, ev, "")
		case ev.GetResourceChanged() != nil:
			expectedEvent := &pb.Event{
				SubscriptionId: u.baseSubID,
				Type: &pb.Event_ResourceChanged{
					ResourceChanged: pbTest.MakeResourceChanged(t, u.deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes,
						ev.GetResourceChanged().GetAuditContext().GetCorrelationId(),
						map[string]interface{}{
							"state": false,
							"power": power,
							"name":  "Light",
						}),
				},
				CorrelationId: "testToken",
			}
			pbTest.CmpEvent(t, expectedEvent, ev, "")
		}
	}
}

type resourceFilter struct {
	httpResourceIdFilter          []string
	resourceIdFilter              []*pb.ResourceIdFilter
	backwardResourceIdFilter      []string
	leadResourceTypeFilterEnabled bool
}

func testRequestHandlerSubscribeToEvents(t *testing.T, deviceID string, resourceFilter resourceFilter) {
	deadline := time.Now().Add(config.TEST_TIMEOUT)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	var opts []service.SetUpOption
	if resourceFilter.leadResourceTypeFilterEnabled {
		coapGWCfg := coapgwTest.MakeConfig(t)
		coapGWCfg.Clients.Eventbus.NATS.LeadResourceType.Enabled = true
		rdCfg := rdTest.MakeConfig(t)
		rdCfg.Clients.Eventbus.NATS.LeadResourceType.Enabled = true
		grpcGWCfg := grpcgwTest.MakeConfig(t)
		grpcGWCfg.Clients.Eventbus.NATS.LeadResourceType.Enabled = true
		raCfg := raTest.MakeConfig(t)
		raCfg.Clients.Eventbus.NATS.LeadResourceType = &natsClient.LeadResourceTypePublisherConfig{
			Enabled:     true,
			RegexFilter: []string{types.CORE_LIGHT},
			UseUUID:     config.LeadResourceUseUUID(),
		}
		err := raCfg.Clients.Eventbus.NATS.Validate()
		require.NoError(t, err)
		opts = append(opts, service.WithCOAPGWConfig(coapGWCfg), service.WithRDConfig(rdCfg), service.WithGRPCGWConfig(grpcGWCfg), service.WithRAConfig(raCfg))
	}
	tearDown := service.SetUp(ctx, t, opts...)
	defer tearDown()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	header := make(http.Header)
	header.Set("Sec-Websocket-Protocol", "Bearer, "+token)
	header.Set("Accept", pkgHttp.ApplicationProtoJsonContentType)
	d := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}

	d.TLSClientConfig = &tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	}
	wsConn, resp, err := d.Dial(fmt.Sprintf("wss://%v/api/v1/ws/events", config.HTTP_GW_HOST), header)
	require.NoError(t, err)
	_ = resp.Body.Close()
	err = wsConn.SetReadDeadline(deadline)
	require.NoError(t, err)

	send := func(req *pb.SubscribeToEvents) error {
		marshaler := runtime.JSONPb{}
		data, errM := marshaler.Marshal(req)
		require.NoError(t, errM)
		return wsConn.WriteMessage(websocket.TextMessage, data)
	}
	sendBackwardResourceIDFilter := func(req *pb.SubscribeToEvents, resourceIDFilter []string) error {
		marshaler := runtime.JSONPb{}
		data, errM := marshaler.Marshal(req)
		require.NoError(t, errM)

		newData, errM := sjson.Delete(string(data), "createSubscription.resourceIdFilter")
		require.NoError(t, errM)
		newData, errM = sjson.Set(newData, "createSubscription.resourceIdFilter", resourceIDFilter)
		require.NoError(t, errM)
		return wsConn.WriteMessage(websocket.TextMessage, []byte(newData))
	}

	recv := func() (*pb.Event, error) {
		_, reader, errM := wsConn.NextReader()
		if errM != nil {
			return nil, errM
		}
		var event pb.Event
		errM = pkgHttpPb.Unmarshal(http.StatusOK, reader, &event)
		return &event, errM
	}

	deviceIDFilter := []string{}
	leadResourceTypeFilter := []string{}
	if resourceFilter.leadResourceTypeFilterEnabled {
		deviceIDFilter = []string{deviceID}
		if config.LeadResourceUseUUID() {
			leadResourceTypeFilter = []string{publisher.ResourceTypeToUUID(types.CORE_LIGHT)}
		} else {
			leadResourceTypeFilter = []string{types.CORE_LIGHT}
		}
	}
	createResourceSub := &pb.SubscribeToEvents{
		CorrelationId: "testToken",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
					pb.SubscribeToEvents_CreateSubscription_REGISTERED,
					pb.SubscribeToEvents_CreateSubscription_UNREGISTERED,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
				},
				ResourceIdFilter:       resourceFilter.resourceIdFilter,
				HttpResourceIdFilter:   resourceFilter.httpResourceIdFilter,
				DeviceIdFilter:         deviceIDFilter,
				LeadResourceTypeFilter: leadResourceTypeFilter,
			},
		},
	}

	if len(resourceFilter.backwardResourceIdFilter) > 0 {
		err = sendBackwardResourceIDFilter(createResourceSub, resourceFilter.backwardResourceIdFilter)
	} else {
		err = send(createResourceSub)
	}
	require.NoError(t, err)

	ev, err := recv()
	require.NoError(t, err)
	expectedEvent := &pb.Event{
		SubscriptionId: ev.GetSubscriptionId(),
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
		CorrelationId: "testToken",
	}
	pbTest.CmpEvent(t, expectedEvent, ev, "")
	baseSubID := ev.GetSubscriptionId()

	deferedCleanUp := true
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	defer func() {
		if deferedCleanUp {
			shutdownDevSim()
		}
	}()

	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: baseSubID,
		Type: &pb.Event_DeviceRegistered_{
			DeviceRegistered: &pb.Event_DeviceRegistered{
				DeviceIds: []string{deviceID},
				EventMetadata: &isEvents.EventMetadata{
					HubId: config.HubID(),
				},
			},
		},
		CorrelationId: "testToken",
	}
	pbTest.CmpEvent(t, expectedEvent, ev, "")

	for {
		ev, err = recv()
		require.NoError(t, err)
		if isDeviceMetadataUpdatedOnlineEvent(ev, deviceID) {
			break
		}
	}
	checkDeviceMetadataUpdatedOnlineEvent(t, ev, deviceID, baseSubID, commands.TwinSynchronization_IN_SYNC)

	err = send(&pb.SubscribeToEvents{
		CorrelationId: "updatePending + resourceUpdated",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				DeviceIdFilter: []string{deviceID},
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATED,
				},
				LeadResourceTypeFilter: leadResourceTypeFilter,
			},
		},
	})
	require.NoError(t, err)

	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.GetSubscriptionId(),
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
		CorrelationId: "updatePending + resourceUpdated",
	}
	pbTest.CmpEvent(t, expectedEvent, ev, "")

	updChecker := &updateChecker{
		c:            c,
		deviceID:     deviceID,
		baseSubID:    baseSubID,
		subUpdatedID: ev.GetSubscriptionId(),
		recv:         recv,
	}
	updChecker.checkUpdateLightResource(ctx, t, 99)
	updChecker.checkUpdateLightResource(ctx, t, 0)

	err = send(&pb.SubscribeToEvents{
		CorrelationId: "receivePending + resourceReceived",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				DeviceIdFilter: []string{deviceID},
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVED,
				},
				LeadResourceTypeFilter: leadResourceTypeFilter,
			},
		},
	})
	require.NoError(t, err)

	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.GetSubscriptionId(),
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
		CorrelationId: "receivePending + resourceReceived",
	}
	pbTest.CmpEvent(t, expectedEvent, ev, "")
	subReceivedID := ev.GetSubscriptionId()

	_, err = c.GetResourceFromDevice(ctx, &pb.GetResourceFromDeviceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
	})
	require.NoError(t, err)
	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: subReceivedID,
		Type: &pb.Event_ResourceRetrievePending{
			ResourceRetrievePending: &events.ResourceRetrievePending{
				ResourceId:    commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
				AuditContext:  ev.GetResourceRetrievePending().GetAuditContext(),
				ResourceTypes: test.TestResourceLightInstanceResourceTypes,
			},
		},
		CorrelationId: "receivePending + resourceReceived",
	}
	pbTest.CmpEvent(t, expectedEvent, ev, "")
	recvCorrelationID := ev.GetResourceRetrievePending().GetAuditContext().GetCorrelationId()

	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: subReceivedID,
		Type: &pb.Event_ResourceRetrieved{
			ResourceRetrieved: pbTest.MakeResourceRetrieved(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, recvCorrelationID,
				map[string]interface{}{
					"name":  "Light",
					"power": 0x0,
					"state": false,
				},
			),
		},
		CorrelationId: "receivePending + resourceReceived",
	}
	pbTest.CmpEvent(t, expectedEvent, ev, "")
	deferedCleanUp = false
	shutdownDevSim()

	run := true
	for run {
		ev, err = recv()
		require.NoError(t, err)

		t.Logf("ev after shutdown: %v\n", ev)

		if ev.GetDeviceUnregistered() != nil {
			expectedEvent = &pb.Event{
				SubscriptionId: ev.GetSubscriptionId(),
				Type: &pb.Event_DeviceUnregistered_{
					DeviceUnregistered: &pb.Event_DeviceUnregistered{
						DeviceIds: []string{deviceID},
						EventMetadata: &isEvents.EventMetadata{
							HubId: config.HubID(),
						},
					},
				},
				CorrelationId: "testToken",
			}
			pbTest.CmpEvent(t, expectedEvent, ev, "")
			run = false
		}
	}
}

func TestRequestHandlerSubscribeToEventsResourceIDFilter(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	testRequestHandlerSubscribeToEvents(t, deviceID, resourceFilter{
		resourceIdFilter: []*pb.ResourceIdFilter{{ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1"))}},
	})
}

func TestRequestHandlerSubscribeToEventsHTTPResourceIDFilter(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	testRequestHandlerSubscribeToEvents(t, deviceID, resourceFilter{
		httpResourceIdFilter: []string{commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")).ToString()},
	})
}

func TestRequestHandlerSubscribeToEventsBackwardResourceIDFilter(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	testRequestHandlerSubscribeToEvents(t, deviceID, resourceFilter{
		backwardResourceIdFilter: []string{commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")).ToString()},
	})
}

func TestRequestHandlerSubscribeToEventsDeviceIDAndLeadResourceTypeFilter(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	testRequestHandlerSubscribeToEvents(t, deviceID, resourceFilter{
		leadResourceTypeFilterEnabled: true,
	})
}
