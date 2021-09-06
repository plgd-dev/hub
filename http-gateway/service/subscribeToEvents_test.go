package service_test

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	coapgwTest "github.com/plgd-dev/cloud/coap-gateway/test"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const TEST_TIMEOUT = time.Second * 30

func TestRequestHandler_SubscribeToEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		sub    *pb.SubscribeToEvents
		accept string
	}
	tests := []struct {
		name string
		args args
		want []*pb.Event
	}{
		{
			name: "invalid - invalid type subscription",
			args: args{
				sub: &pb.SubscribeToEvents{
					CorrelationId: "testToken",
				},
				accept: uri.ApplicationProtoJsonContentType,
			},

			want: []*pb.Event{
				{
					Type: &pb.Event_OperationProcessed_{
						OperationProcessed: &pb.Event_OperationProcessed{
							ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
								Code:    pb.Event_OperationProcessed_ErrorStatus_ERROR,
								Message: "invalid action('<nil>')",
							},
						},
					},
					CorrelationId: "testToken",
				},
			},
		},
		{
			name: "devices subscription - registered",
			args: args{
				sub: &pb.SubscribeToEvents{
					CorrelationId: "testToken",
					Action: &pb.SubscribeToEvents_CreateSubscription_{
						CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
							EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
								pb.SubscribeToEvents_CreateSubscription_REGISTERED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED,
							},
							IncludeCurrentState: true,
						},
					},
				},
			},
			want: []*pb.Event{
				{
					Type: &pb.Event_OperationProcessed_{
						OperationProcessed: &pb.Event_OperationProcessed{
							ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
								Code: pb.Event_OperationProcessed_ErrorStatus_OK,
							},
						},
					},
					CorrelationId: "testToken",
				},
				{
					Type: &pb.Event_DeviceRegistered_{
						DeviceRegistered: &pb.Event_DeviceRegistered{
							DeviceIds: []string{deviceID},
						},
					},
					CorrelationId: "testToken",
				},
			},
		},
		{
			name: "devices subscription - online",
			args: args{
				sub: &pb.SubscribeToEvents{
					CorrelationId: "testToken",
					Action: &pb.SubscribeToEvents_CreateSubscription_{
						CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
							EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
								pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
							},
							IncludeCurrentState: true,
						},
					},
				},
				accept: uri.ApplicationProtoJsonContentType,
			},
			want: []*pb.Event{
				{
					Type: &pb.Event_OperationProcessed_{
						OperationProcessed: &pb.Event_OperationProcessed{
							ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
								Code: pb.Event_OperationProcessed_ErrorStatus_OK,
							},
						},
					},
					CorrelationId: "testToken",
				},
				{
					Type: &pb.Event_DeviceMetadataUpdated{
						DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
							DeviceId: deviceID,
							Status: &commands.ConnectionStatus{
								Value: commands.ConnectionStatus_ONLINE,
							},
						},
					},
					CorrelationId: "testToken",
				},
			},
		},
		{
			name: "device subscription - published",
			args: args{
				sub: &pb.SubscribeToEvents{
					CorrelationId: "testToken",
					Action: &pb.SubscribeToEvents_CreateSubscription_{
						CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
							DeviceIdFilter: []string{deviceID},
							EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
								pb.SubscribeToEvents_CreateSubscription_RESOURCE_PUBLISHED, pb.SubscribeToEvents_CreateSubscription_RESOURCE_UNPUBLISHED,
							},
							IncludeCurrentState: true,
						},
					},
				},
				accept: uri.ApplicationProtoJsonContentType,
			},
			want: []*pb.Event{
				{
					Type: &pb.Event_OperationProcessed_{
						OperationProcessed: &pb.Event_OperationProcessed{
							ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
								Code: pb.Event_OperationProcessed_ErrorStatus_OK,
							},
						},
					},
					CorrelationId: "testToken",
				},
				test.ResourceLinkToPublishEvent(deviceID, "testToken", test.GetAllBackendResourceLinks()),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := make(http.Header)
			header.Set("Sec-Websocket-Protocol", "Bearer, "+token)
			d := &websocket.Dialer{
				Proxy:            http.ProxyFromEnvironment,
				HandshakeTimeout: 45 * time.Second,
			}

			d.TLSClientConfig = &tls.Config{
				RootCAs: test.GetRootCertificatePool(t),
			}
			wsConn, _, err := d.Dial(fmt.Sprintf("wss://%v/api/v1/ws/events?accept=%v", testCfg.HTTP_GW_HOST, tt.args.accept), header)
			require.NoError(t, err)
			defer func() {
				_ = wsConn.Close()
			}()

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

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				for _, w := range tt.want {
					ev, err := recv()
					require.NoError(t, err)
					ev.SubscriptionId = w.SubscriptionId
					if ev.GetResourcePublished() != nil {
						test.CleanUpResourceLinksPublished(ev.GetResourcePublished())
					}
					if w.GetResourcePublished() != nil {
						test.CleanUpResourceLinksPublished(w.GetResourcePublished())
					}
					if ev.GetDeviceMetadataUpdated() != nil {
						ev.GetDeviceMetadataUpdated().EventMetadata = nil
						ev.GetDeviceMetadataUpdated().AuditContext = nil
						if ev.GetDeviceMetadataUpdated().GetStatus() != nil {
							ev.GetDeviceMetadataUpdated().GetStatus().ValidUntil = 0
						}
					}
					test.CheckProtobufs(t, tt.want, ev, test.RequireToCheckFunc(require.Contains))
				}
			}()
			err = send(tt.args.sub)
			require.NoError(t, err)
			wg.Wait()
		})
	}
}

func TestRequestHandler_ValidateEventsFlow(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()

	coapgwCfg := coapgwTest.MakeConfig(t)

	tearDown := test.SetUp(ctx, t, test.WithCOAPGWConfig(coapgwCfg))
	defer tearDown()

	token := oauthTest.GetServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())

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
	wsConn, _, err := d.Dial(fmt.Sprintf("wss://%v/api/v1/ws/events", testCfg.HTTP_GW_HOST), header)
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
					pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED, pb.SubscribeToEvents_CreateSubscription_REGISTERED, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED,
				},
				IncludeCurrentState: true,
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

	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
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
		if ev.GetDeviceMetadataUpdated() != nil && ev.GetDeviceMetadataUpdated().GetDeviceId() == deviceID && ev.GetDeviceMetadataUpdated().GetStatus().GetValue() == commands.ConnectionStatus_ONLINE {
			break
		}
		continue
	}
	if ev.GetDeviceMetadataUpdated() != nil {
		ev.GetDeviceMetadataUpdated().EventMetadata = nil
		ev.GetDeviceMetadataUpdated().AuditContext = nil
		if ev.GetDeviceMetadataUpdated().GetStatus() != nil {
			ev.GetDeviceMetadataUpdated().GetStatus().ValidUntil = 0
		}
	}
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				DeviceId: deviceID,
				Status: &commands.ConnectionStatus{
					Value: commands.ConnectionStatus_ONLINE,
				},
			},
		},
		CorrelationId: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	err = send(&pb.SubscribeToEvents{
		CorrelationId: "testToken",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				ResourceIdFilter: []string{commands.NewResourceID(deviceID, "/light/2").ToString()},
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
				},
				IncludeCurrentState: true,
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
		CorrelationId: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
	subContentChangedID := ev.SubscriptionId

	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: subContentChangedID,
		Type: &pb.Event_ResourceChanged{
			ResourceChanged: &events.ResourceChanged{
				ResourceId: commands.NewResourceID(deviceID, "/light/2"),
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
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING, pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATED,
				},
				IncludeCurrentState: true,
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
	subUpdatedID := ev.SubscriptionId

	_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, "/light/2"),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: func() []byte {
				v := map[string]interface{}{
					"power": 99,
				}
				d, err := cbor.Encode(v)
				require.NoError(t, err)
				return d
			}(),
		},
	})
	require.NoError(t, err)

	var updCorrelationID string
	for i := 0; i < 3; i++ {
		ev, err = recv()
		require.NoError(t, err)
		switch {
		case ev.GetResourceUpdatePending() != nil:
			expectedEvent = &pb.Event{
				SubscriptionId: subUpdatedID,
				Type: &pb.Event_ResourceUpdatePending{
					ResourceUpdatePending: &events.ResourceUpdatePending{
						ResourceId: commands.NewResourceID(deviceID, "/light/2"),
						Content: &commands.Content{
							ContentType:       message.AppOcfCbor.String(),
							CoapContentFormat: -1,
							Data: func() []byte {
								v := map[string]interface{}{
									"power": 99,
								}
								d, err := cbor.Encode(v)
								require.NoError(t, err)
								return d
							}(),
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
			expectedEvent = &pb.Event{
				SubscriptionId: subUpdatedID,
				Type: &pb.Event_ResourceUpdated{
					ResourceUpdated: &events.ResourceUpdated{
						ResourceId:    commands.NewResourceID(deviceID, "/light/2"),
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
			expectedEvent = &pb.Event{
				SubscriptionId: subContentChangedID,
				Type: &pb.Event_ResourceChanged{
					ResourceChanged: &events.ResourceChanged{
						ResourceId: commands.NewResourceID(deviceID, "/light/2"),
						Content: &commands.Content{
							CoapContentFormat: int32(message.AppOcfCbor),
							ContentType:       message.AppOcfCbor.String(),
							Data:              []byte("\277estate\364epower\030cdnameeLight\377"),
						},
						Status:        commands.Status_OK,
						AuditContext:  ev.GetResourceChanged().GetAuditContext(),
						EventMetadata: ev.GetResourceChanged().GetEventMetadata(),
					},
				},
				CorrelationId: "testToken",
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
		}
	}
	_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, "/light/2"),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: func() []byte {
				v := map[string]interface{}{
					"power": 0,
				}
				d, err := cbor.Encode(v)
				require.NoError(t, err)
				return d
			}(),
		},
	})
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		ev, err = recv()
		require.NoError(t, err)
		switch {
		case ev.GetResourceUpdatePending() != nil:
			expectedEvent = &pb.Event{
				SubscriptionId: subUpdatedID,
				Type: &pb.Event_ResourceUpdatePending{
					ResourceUpdatePending: &events.ResourceUpdatePending{
						ResourceId: commands.NewResourceID(deviceID, "/light/2"),
						Content: &commands.Content{
							ContentType:       message.AppOcfCbor.String(),
							CoapContentFormat: -1,
							Data: func() []byte {
								v := map[string]interface{}{
									"power": 0,
								}
								d, err := cbor.Encode(v)
								require.NoError(t, err)
								return d
							}(),
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
			expectedEvent = &pb.Event{
				SubscriptionId: subUpdatedID,
				Type: &pb.Event_ResourceUpdated{
					ResourceUpdated: &events.ResourceUpdated{
						ResourceId:    commands.NewResourceID(deviceID, "/light/2"),
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
			expectedEvent = &pb.Event{
				SubscriptionId: subContentChangedID,
				Type: &pb.Event_ResourceChanged{
					ResourceChanged: &events.ResourceChanged{
						ResourceId: commands.NewResourceID(deviceID, "/light/2"),
						Content: &commands.Content{
							CoapContentFormat: int32(message.AppOcfCbor),
							ContentType:       message.AppOcfCbor.String(),
							Data:              []byte("\277estate\364epower\000dnameeLight\377"),
						},
						Status:        commands.Status_OK,
						AuditContext:  ev.GetResourceChanged().GetAuditContext(),
						EventMetadata: ev.GetResourceChanged().GetEventMetadata(),
					},
				},
				CorrelationId: "testToken",
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
		}
	}

	err = send(&pb.SubscribeToEvents{
		CorrelationId: "receivePending + resourceReceived",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				DeviceIdFilter: []string{deviceID},
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING, pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVED,
				},
				IncludeCurrentState: true,
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
		ResourceId: commands.NewResourceID(deviceID, "/light/2"),
	})
	require.NoError(t, err)
	ev, err = recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: subReceivedID,
		Type: &pb.Event_ResourceRetrievePending{
			ResourceRetrievePending: &events.ResourceRetrievePending{
				ResourceId:    commands.NewResourceID(deviceID, "/light/2"),
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
				ResourceId: commands.NewResourceID(deviceID, "/light/2"),
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
