package service_test

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/http-gateway/test"
	"github.com/plgd-dev/hub/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthService "github.com/plgd-dev/hub/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/test/pb"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func isDeviceMetadataUpdatedOnlineEvent(ev *pb.Event, deviceID string) bool {
	return ev.GetDeviceMetadataUpdated() != nil &&
		ev.GetDeviceMetadataUpdated().GetDeviceId() == deviceID &&
		ev.GetDeviceMetadataUpdated().GetStatus().GetValue() == commands.ConnectionStatus_ONLINE
}

func checkDeviceMetadataUpdatedOnlineEvent(t *testing.T, ev *pb.Event, deviceID, baseSubId string) {
	expectedEvent := &pb.Event{
		SubscriptionId: baseSubId,
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				DeviceId: deviceID,
				Status: &commands.ConnectionStatus{
					Value: commands.ConnectionStatus_ONLINE,
				},
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
			},
		},
		CorrelationId: "testToken",
	}
	pbTest.CmpEvent(t, expectedEvent, ev)
}

type updateChecker struct {
	c            pb.GrpcGatewayClient
	deviceID     string
	baseSubId    string
	subUpdatedID string

	recv func() (*pb.Event, error)
}

// update light resource and check received events
func (u *updateChecker) checkUpdateLightResource(t *testing.T, ctx context.Context, power uint64) {
	_, err := u.c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(u.deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[interface{}]interface{}{
				"power": power,
			}),
		},
	})
	require.NoError(t, err)

	var updCorrelationID string
	for i := 0; i < 3; i++ {
		ev, err := u.recv()
		require.NoError(t, err)
		switch {
		case ev.GetResourceUpdatePending() != nil:
			expectedEvent := &pb.Event{
				SubscriptionId: u.subUpdatedID,
				Type: &pb.Event_ResourceUpdatePending{
					ResourceUpdatePending: &events.ResourceUpdatePending{
						ResourceId: commands.NewResourceID(u.deviceID, test.TestResourceLightInstanceHref("1")),
						Content: &commands.Content{
							ContentType:       message.AppOcfCbor.String(),
							CoapContentFormat: -1,
							Data: test.EncodeToCbor(t, map[interface{}]interface{}{
								"power": power,
							}),
						},
						AuditContext:  ev.GetResourceUpdatePending().GetAuditContext(),
						EventMetadata: ev.GetResourceUpdatePending().GetEventMetadata(),
					},
				},
				CorrelationId: "updatePending + resourceUpdated",
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
			updCorrelationID = ev.GetResourceUpdatePending().GetAuditContext().GetCorrelationId()
		case ev.GetResourceUpdated() != nil:
			expectedEvent := &pb.Event{
				SubscriptionId: u.subUpdatedID,
				Type: &pb.Event_ResourceUpdated{
					ResourceUpdated: &events.ResourceUpdated{
						ResourceId:    commands.NewResourceID(u.deviceID, test.TestResourceLightInstanceHref("1")),
						Status:        commands.Status_OK,
						Content:       ev.GetResourceUpdated().GetContent(),
						AuditContext:  commands.NewAuditContext(ev.GetResourceUpdated().GetAuditContext().GetUserId(), updCorrelationID),
						EventMetadata: ev.GetResourceUpdated().GetEventMetadata(),
					},
				},
				CorrelationId: "updatePending + resourceUpdated",
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
		case ev.GetResourceChanged() != nil:
			expData := map[interface{}]interface{}{
				"state": false,
				"power": power,
				"name":  "Light",
			}
			expectedEvent := &pb.Event{
				SubscriptionId: u.baseSubId,
				Type: &pb.Event_ResourceChanged{
					ResourceChanged: &events.ResourceChanged{
						ResourceId: commands.NewResourceID(u.deviceID, test.TestResourceLightInstanceHref("1")),
						Content: &commands.Content{
							CoapContentFormat: int32(message.AppOcfCbor),
							ContentType:       message.AppOcfCbor.String(),
							Data:              nil,
						},
						Status:        commands.Status_OK,
						AuditContext:  ev.GetResourceChanged().GetAuditContext(),
						EventMetadata: ev.GetResourceChanged().GetEventMetadata(),
					},
				},
				CorrelationId: "testToken",
			}
			data := test.DecodeCbor(t, ev.GetResourceChanged().GetContent().GetData())
			require.Equal(t, expData, data)
			ev.GetResourceChanged().GetContent().Data = nil
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
		}
	}
}

func TestRequestHandlerSubscribeToEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	header := make(http.Header)
	header.Set("Sec-Websocket-Protocol", "Bearer, "+token)
	header.Set("Accept", uri.ApplicationProtoJsonContentType)
	d := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}

	d.TLSClientConfig = &tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	}
	wsConn, _, err := d.Dial(fmt.Sprintf("wss://%v/api/v1/ws/events", config.HTTP_GW_HOST), header)
	require.NoError(t, err)

	send := func(req *pb.SubscribeToEvents) error {
		marshaler := runtime.JSONPb{}
		data, err := marshaler.Marshal(req)
		require.NoError(t, err)
		return wsConn.WriteMessage(websocket.TextMessage, data)
	}

	recv := func() (*pb.Event, error) {
		_, reader, err := wsConn.NextReader()
		if err != nil {
			return nil, err
		}
		var event pb.Event
		err = Unmarshal(http.StatusOK, reader, &event)
		return &event, err
	}

	err = send(&pb.SubscribeToEvents{
		CorrelationId: "testToken",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
					pb.SubscribeToEvents_CreateSubscription_REGISTERED,
					pb.SubscribeToEvents_CreateSubscription_UNREGISTERED,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
				},
				ResourceIdFilter: []string{commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")).ToString()},
			},
		},
	})
	require.NoError(t, err)

	ev, err := recv()
	require.NoError(t, err)
	expectedEvent := &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
		CorrelationId: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
	baseSubId := ev.SubscriptionId

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, nil)

	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: baseSubId,
		Type: &pb.Event_DeviceRegistered_{
			DeviceRegistered: &pb.Event_DeviceRegistered{
				DeviceIds: []string{deviceID},
			},
		},
		CorrelationId: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	for {
		ev, err = recv()
		require.NoError(t, err)
		if isDeviceMetadataUpdatedOnlineEvent(ev, deviceID) {
			break
		}
	}
	checkDeviceMetadataUpdatedOnlineEvent(t, ev, deviceID, baseSubId)

	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: baseSubId,
		Type: &pb.Event_ResourceChanged{
			ResourceChanged: &events.ResourceChanged{
				ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
				Content: &commands.Content{
					CoapContentFormat: int32(message.AppOcfCbor),
					ContentType:       message.AppOcfCbor.String(),
					Data: func() []byte {
						ret, err := base64.StdEncoding.DecodeString("v2JydJ9qY29yZS5saWdodP9iaWafaW9pYy5pZi5yd29vaWMuaWYuYmFzZWxpbmX/ZXN0YXRl9GVwb3dlcgBkbmFtZWVMaWdodP8=")
						require.NoError(t, err)
						return ret
					}(),
				},
				Status:        commands.Status_OK,
				AuditContext:  ev.GetResourceChanged().GetAuditContext(),
				EventMetadata: ev.GetResourceChanged().GetEventMetadata(),
			},
		},
		CorrelationId: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	err = send(&pb.SubscribeToEvents{
		CorrelationId: "updatePending + resourceUpdated",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				DeviceIdFilter: []string{deviceID},
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATED,
				},
			},
		},
	})
	require.NoError(t, err)

	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
		CorrelationId: "updatePending + resourceUpdated",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	updChecker := &updateChecker{
		c:            c,
		deviceID:     deviceID,
		baseSubId:    baseSubId,
		subUpdatedID: ev.SubscriptionId,
		recv:         recv,
	}
	updChecker.checkUpdateLightResource(t, ctx, 99)
	updChecker.checkUpdateLightResource(t, ctx, 0)

	err = send(&pb.SubscribeToEvents{
		CorrelationId: "receivePending + resourceReceived",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				DeviceIdFilter: []string{deviceID},
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING, pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVED,
				},
			},
		},
	})
	require.NoError(t, err)

	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
		CorrelationId: "receivePending + resourceReceived",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
	subReceivedID := ev.SubscriptionId

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
				EventMetadata: ev.GetResourceRetrievePending().GetEventMetadata(),
			},
		},
		CorrelationId: "receivePending + resourceReceived",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
	recvCorrelationID := ev.GetResourceRetrievePending().GetAuditContext().GetCorrelationId()

	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: subReceivedID,
		Type: &pb.Event_ResourceRetrieved{
			ResourceRetrieved: &events.ResourceRetrieved{
				ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
				Content: &commands.Content{
					ContentType:       message.AppOcfCbor.String(),
					CoapContentFormat: int32(message.AppOcfCbor),
					Data:              []byte("\277estate\364epower\000dnameeLight\377"),
				},
				Status:        commands.Status_OK,
				AuditContext:  commands.NewAuditContext(ev.GetResourceRetrieved().GetAuditContext().GetUserId(), recvCorrelationID),
				EventMetadata: ev.GetResourceRetrieved().GetEventMetadata(),
			},
		},
		CorrelationId: "receivePending + resourceReceived",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	shutdownDevSim()

	run := true
	for run {
		ev, err = recv()
		require.NoError(t, err)

		t.Logf("ev after shutdown: %v\n", ev)

		switch {
		case ev.GetDeviceUnregistered() != nil:
			expectedEvent = &pb.Event{
				SubscriptionId: ev.SubscriptionId,
				Type: &pb.Event_DeviceUnregistered_{
					DeviceUnregistered: &pb.Event_DeviceUnregistered{
						DeviceIds: []string{deviceID},
					},
				},
				CorrelationId: "testToken",
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
			run = false
		}
	}
}
