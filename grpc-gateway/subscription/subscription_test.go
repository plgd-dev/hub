package subscription_test

import (
	"context"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/stretchr/testify/require"

	"github.com/plgd-dev/cloud/authorization/client"
	authpb "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	subscription "github.com/plgd-dev/cloud/grpc-gateway/subscription"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	natsTest "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	raservice "github.com/plgd-dev/cloud/resource-aggregate/service"
	"github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
)

func TestRequestHandler_SubscribeToEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		sub *pb.SubscribeToEvents
	}
	tests := []struct {
		name string
		args args
		want []*pb.Event
	}{
		{
			name: "all events - include current state",
			args: args{
				sub: &pb.SubscribeToEvents{
					CorrelationId: "testToken",
					Action: &pb.SubscribeToEvents_CreateSubscription_{
						CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
							IncludeCurrentState: true,
						},
					},
				},
			},
			want: []*pb.Event{
				{
					Type: &pb.Event_DeviceRegistered_{
						DeviceRegistered: &pb.Event_DeviceRegistered{
							DeviceIds: []string{deviceID},
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
				test.ResourceLinkToPublishEvent(deviceID, "testToken", test.GetAllBackendResourceLinks()),
				{
					Type: &pb.Event_ResourceChanged{
						ResourceChanged: &events.ResourceChanged{
							ResourceId: commands.NewResourceID(deviceID, ""),
							Status:     commands.Status_OK,
						},
					},
					CorrelationId: "testToken",
				},
				{
					Type: &pb.Event_ResourceChanged{
						ResourceChanged: &events.ResourceChanged{
							ResourceId: commands.NewResourceID(deviceID, ""),
							Status:     commands.Status_OK,
						},
					},
					CorrelationId: "testToken",
				},
				{
					Type: &pb.Event_ResourceChanged{
						ResourceChanged: &events.ResourceChanged{
							ResourceId: commands.NewResourceID(deviceID, ""),
							Status:     commands.Status_OK,
						},
					},
					CorrelationId: "testToken",
				},
				{
					Type: &pb.Event_ResourceChanged{
						ResourceChanged: &events.ResourceChanged{
							ResourceId: commands.NewResourceID(deviceID, ""),
							Status:     commands.Status_OK,
						},
					},
					CorrelationId: "testToken",
				},
				{
					Type: &pb.Event_ResourceChanged{
						ResourceChanged: &events.ResourceChanged{
							ResourceId: commands.NewResourceID(deviceID, ""),
							Status:     commands.Status_OK,
						},
					},
					CorrelationId: "testToken",
				},
			},
		},
		{
			name: "all events - without current state",
			args: args{
				sub: &pb.SubscribeToEvents{
					CorrelationId: "testToken",
				},
			},
			want: []*pb.Event{},
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
			},
			want: []*pb.Event{
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
			},
			want: []*pb.Event{

				test.ResourceLinkToPublishEvent(deviceID, "testToken", test.GetAllBackendResourceLinks()),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	token := oauthTest.GetServiceToken(t)
	ctx = kitNetGrpc.CtxWithIncomingToken(kitNetGrpc.CtxWithToken(ctx, token), token)

	rdConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST), log.Get())
	require.NoError(t, err)
	defer func() {
		_ = rdConn.Close()
	}()
	rdc := pb.NewGrpcGatewayClient(rdConn.GRPC())

	asConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.AUTH_HOST), log.Get())
	require.NoError(t, err)
	defer func() {
		_ = asConn.Close()
	}()
	asc := authpb.NewAuthorizationServiceClient(asConn.GRPC())

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, rdc, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	pool, err := ants.NewPool(1)
	require.NoError(t, err)
	natsConn, resourceSubscriber, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), log.Get(), subscriber.WithGoPool(pool.Submit), subscriber.WithUnmarshaler(utils.Unmarshal))
	require.NoError(t, err)
	defer natsConn.Close()
	defer resourceSubscriber.Close()

	ownerClaim := "sub"

	ownerCache := client.NewOwnerCache(ownerClaim, time.Minute, resourceSubscriber.Conn(), asc, func(err error) { t.Log(err) })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result []*pb.Event
			s := subscription.New(ctx, resourceSubscriber, rdc, func(e *pb.Event) error {
				result = append(result, e)
				return nil
			}, tt.args.sub.GetCorrelationId(), 0, time.Second, func(err error) { t.Log(err) }, tt.args.sub.GetCreateSubscription())
			err := s.Init(ownerCache)
			require.NoError(t, err)
			defer func() {
				err := s.Close()
				require.NoError(t, err)
			}()
			require.Equal(t, len(tt.want), len(result))

			for i, ev := range result {
				require.NotEmpty(t, ev.SubscriptionId)
				ev.SubscriptionId = tt.want[i].SubscriptionId
				if ev.GetResourcePublished() != nil {
					test.CleanUpResourceLinksPublished(ev.GetResourcePublished())
				}
				if tt.want[i].GetResourcePublished() != nil {
					test.CleanUpResourceLinksPublished(tt.want[i].GetResourcePublished())
				}
				if ev.GetDeviceMetadataUpdated() != nil {
					ev.GetDeviceMetadataUpdated().EventMetadata = nil
					ev.GetDeviceMetadataUpdated().AuditContext = nil
					if ev.GetDeviceMetadataUpdated().GetStatus() != nil {
						ev.GetDeviceMetadataUpdated().GetStatus().ValidUntil = 0
					}
				}
				if ev.GetResourceChanged() != nil {
					require.NotEmpty(t, ev.GetResourceChanged().GetEventMetadata())
					ev.GetResourceChanged().EventMetadata = nil
					require.NotEmpty(t, ev.GetResourceChanged().GetAuditContext())
					ev.GetResourceChanged().AuditContext = nil
					require.NotEmpty(t, ev.GetResourceChanged().GetResourceId().GetHref())
					ev.GetResourceChanged().GetResourceId().Href = ""
					require.NotEmpty(t, ev.GetResourceChanged().GetContent())
					ev.GetResourceChanged().Content = nil
				}
				test.CheckProtobufs(t, tt.want[i], ev, test.RequireToCheckFunc(require.Equal))
			}

		})
	}
}

func waitForEvent(ctx context.Context, t *testing.T, recvChan <-chan *pb.Event) *pb.Event {
	select {
	case ev := <-recvChan:
		return ev
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}
	return nil
}

func check(t *testing.T, ev *pb.Event, expectedEvent *pb.Event) {
	if ev.GetResourcePublished() != nil {
		test.CleanUpResourceLinksPublished(ev.GetResourcePublished())
		ev.GetResourcePublished().AuditContext = nil
	}
	if expectedEvent.GetResourcePublished() != nil {
		expectedEvent.SubscriptionId = ev.SubscriptionId
		test.CleanUpResourceLinksPublished(expectedEvent.GetResourcePublished())
	}
	if ev.GetDeviceMetadataUpdated() != nil {
		ev.GetDeviceMetadataUpdated().EventMetadata = nil
		ev.GetDeviceMetadataUpdated().AuditContext = nil
		if ev.GetDeviceMetadataUpdated().GetStatus() != nil {
			ev.GetDeviceMetadataUpdated().GetStatus().ValidUntil = 0
		}
	}
	if ev.GetResourceChanged() != nil {
		require.NotEmpty(t, ev.GetResourceChanged().GetEventMetadata())
		ev.GetResourceChanged().EventMetadata = nil
		require.NotEmpty(t, ev.GetResourceChanged().GetAuditContext())
		ev.GetResourceChanged().AuditContext = nil
		require.NotEmpty(t, ev.GetResourceChanged().GetResourceId().GetHref())
		ev.GetResourceChanged().GetResourceId().Href = ""
		require.NotEmpty(t, ev.GetResourceChanged().GetContent().GetData())
		ev.GetResourceChanged().GetContent().Data = nil
	}
	if ev.GetResourceUpdatePending() != nil {
		require.NotEmpty(t, ev.GetResourceUpdatePending().GetEventMetadata())
		ev.GetResourceUpdatePending().EventMetadata = nil
		require.NotEmpty(t, ev.GetResourceUpdatePending().GetAuditContext())
		ev.GetResourceUpdatePending().AuditContext = nil
	}
	if ev.GetResourceUpdated() != nil {
		require.NotEmpty(t, ev.GetResourceUpdated().GetEventMetadata())
		ev.GetResourceUpdated().EventMetadata = nil
		require.NotEmpty(t, ev.GetResourceUpdated().GetAuditContext())
		ev.GetResourceUpdated().AuditContext = nil
	}
	if ev.GetResourceRetrievePending() != nil {
		require.NotEmpty(t, ev.GetResourceRetrievePending().GetEventMetadata())
		ev.GetResourceRetrievePending().EventMetadata = nil
		require.NotEmpty(t, ev.GetResourceRetrievePending().GetAuditContext())
		ev.GetResourceRetrievePending().AuditContext = nil
	}
	if ev.GetResourceRetrieved() != nil {
		require.NotEmpty(t, ev.GetResourceRetrieved().GetEventMetadata())
		ev.GetResourceRetrieved().EventMetadata = nil
		require.NotEmpty(t, ev.GetResourceRetrieved().GetAuditContext())
		ev.GetResourceRetrieved().AuditContext = nil
		require.NotEmpty(t, ev.GetResourceRetrieved().GetContent().GetData())
		ev.GetResourceRetrieved().GetContent().Data = nil
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
}

func checkAndValidateUpdate(ctx context.Context, t *testing.T, rac raservice.ResourceAggregateClient, s *subscription.Sub, recvChan <-chan *pb.Event, correlationID string, deviceID string, value uint64) {
	updCorrelationID := "updCorrelationID"
	_, err := rac.UpdateResource(ctx, &commands.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, "/light/2"),
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

	check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_ResourceUpdatePending{
			ResourceUpdatePending: &events.ResourceUpdatePending{
				ResourceId: commands.NewResourceID(deviceID, "/light/2"),
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
			},
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
					ResourceUpdated: &events.ResourceUpdated{
						ResourceId: commands.NewResourceID(deviceID, "/light/2"),
						Content: &commands.Content{
							CoapContentFormat: -1,
						},
						Status: commands.Status_OK,
					},
				},
				CorrelationId: correlationID,
			})
		case ev.GetResourceChanged() != nil:
			check(t, ev, &pb.Event{
				SubscriptionId: s.Id(),
				Type: &pb.Event_ResourceChanged{
					ResourceChanged: &events.ResourceChanged{
						ResourceId: commands.NewResourceID(deviceID, ""),
						Content: &commands.Content{
							ContentType:       message.AppOcfCbor.String(),
							CoapContentFormat: 10000,
						},
						Status: commands.Status_OK,
					},
				},
				CorrelationId: correlationID,
			})
		}
	}
}

func checkAndValidateRetrieve(ctx context.Context, t *testing.T, rac raservice.ResourceAggregateClient, s *subscription.Sub, recvChan <-chan *pb.Event, correlationID string, deviceID string) {
	retrieveCorrelationID := "retrieveCorrelationID"
	_, err := rac.RetrieveResource(ctx, &commands.RetrieveResourceRequest{
		ResourceId:    commands.NewResourceID(deviceID, "/light/2"),
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
				ResourceId: commands.NewResourceID(deviceID, "/light/2"),
			},
		},
		CorrelationId: correlationID,
	})
	check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_ResourceRetrieved{
			ResourceRetrieved: &events.ResourceRetrieved{
				ResourceId: commands.NewResourceID(deviceID, "/light/2"),
				Content: &commands.Content{
					CoapContentFormat: int32(message.AppOcfCbor),
					ContentType:       message.AppOcfCbor.String(),
				},
				Status: commands.Status_OK,
			},
		},
		CorrelationId: correlationID,
	})
}

func TestRequestHandler_ValidateEventsFlow(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	token := oauthTest.GetServiceToken(t)
	ctx = kitNetGrpc.CtxWithIncomingToken(kitNetGrpc.CtxWithToken(ctx, token), token)

	rdConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST), log.Get())
	require.NoError(t, err)
	defer func() {
		_ = rdConn.Close()
	}()
	rdc := pb.NewGrpcGatewayClient(rdConn.GRPC())

	raConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST), log.Get())
	require.NoError(t, err)
	defer func() {
		_ = raConn.Close()
	}()
	rac := raservice.NewResourceAggregateClient(raConn.GRPC())

	asConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.AUTH_HOST), log.Get())
	require.NoError(t, err)
	defer func() {
		_ = asConn.Close()
	}()
	asc := authpb.NewAuthorizationServiceClient(asConn.GRPC())

	pool, err := ants.NewPool(1)
	require.NoError(t, err)
	natsConn, resourceSubscriber, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), log.Get(), subscriber.WithGoPool(pool.Submit), subscriber.WithUnmarshaler(utils.Unmarshal))
	require.NoError(t, err)
	defer natsConn.Close()
	defer resourceSubscriber.Close()

	ownerClaim := "sub"

	ownerCache := client.NewOwnerCache(ownerClaim, time.Minute, resourceSubscriber.Conn(), asc, func(err error) { t.Log(err) })
	correlationID := "testToken"
	recvChan := make(chan *pb.Event, 1)

	s := subscription.New(ctx, resourceSubscriber, rdc, func(e *pb.Event) error {
		select {
		case recvChan <- e:
		case <-ctx.Done():
		}
		return nil
	}, correlationID, 10, time.Second, func(err error) { t.Log(err) }, &pb.SubscribeToEvents_CreateSubscription{})
	err = s.Init(ownerCache)
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
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				DeviceId: deviceID,
				Status: &commands.ConnectionStatus{
					Value: commands.ConnectionStatus_ONLINE,
				},
			},
		},
		CorrelationId: correlationID,
	})
	check(t, waitForEvent(ctx, t, recvChan), test.ResourceLinkToPublishEvent(deviceID, correlationID, test.GetAllBackendResourceLinks()))

	for range test.GetAllBackendResourceLinks() {
		check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
			SubscriptionId: s.Id(),
			Type: &pb.Event_ResourceChanged{
				ResourceChanged: &events.ResourceChanged{
					ResourceId: commands.NewResourceID(deviceID, ""),
					Content: &commands.Content{
						CoapContentFormat: int32(message.AppOcfCbor),
						ContentType:       message.AppOcfCbor.String(),
					},
					Status: commands.Status_OK,
				},
			},
			CorrelationId: correlationID,
		})
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
