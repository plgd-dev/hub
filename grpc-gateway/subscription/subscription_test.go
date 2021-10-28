package subscription_test

import (
	"context"
	"testing"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	subscription "github.com/plgd-dev/hub/grpc-gateway/subscription"
	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/subscriber"
	natsTest "github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	raservice "github.com/plgd-dev/hub/resource-aggregate/service"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthService "github.com/plgd-dev/hub/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/test/pb"
	"github.com/plgd-dev/hub/test/service"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/stretchr/testify/require"
)

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
		pbTest.CleanUpResourceLinksPublished(ev.GetResourcePublished())
	}
	if expectedEvent.GetResourcePublished() != nil {
		expectedEvent.SubscriptionId = ev.SubscriptionId
		pbTest.CleanUpResourceLinksPublished(expectedEvent.GetResourcePublished())
	}
	if ev.GetDeviceMetadataUpdated() != nil {
		pbTest.CleanUpDeviceMetadataUpdated(ev.GetDeviceMetadataUpdated())
	}
	if ev.GetResourceChanged() != nil {
		require.NotEmpty(t, ev.GetResourceChanged().GetEventMetadata())
		require.NotEmpty(t, ev.GetResourceChanged().GetAuditContext())
		pbTest.CleanUpResourceChanged(ev.GetResourceChanged())
		require.NotEmpty(t, ev.GetResourceChanged().GetResourceId().GetHref())
		ev.GetResourceChanged().GetResourceId().Href = ""
		require.NotEmpty(t, ev.GetResourceChanged().GetContent().GetData())
		ev.GetResourceChanged().GetContent().Data = nil
	}
	if ev.GetResourceUpdatePending() != nil {
		require.NotEmpty(t, ev.GetResourceUpdatePending().GetEventMetadata())
		require.NotEmpty(t, ev.GetResourceUpdatePending().GetAuditContext())
		pbTest.CleanUpResourceUpdatePending(ev.GetResourceUpdatePending())
	}
	if ev.GetResourceUpdated() != nil {
		require.NotEmpty(t, ev.GetResourceUpdated().GetEventMetadata())
		require.NotEmpty(t, ev.GetResourceUpdated().GetAuditContext())
		pbTest.CleanUpResourceUpdated(ev.GetResourceUpdated())
	}
	if ev.GetResourceRetrievePending() != nil {
		require.NotEmpty(t, ev.GetResourceRetrievePending().GetEventMetadata())
		require.NotEmpty(t, ev.GetResourceRetrievePending().GetAuditContext())
		pbTest.CleanUpResourceRetrievePending(ev.GetResourceRetrievePending())
	}
	if ev.GetResourceRetrieved() != nil {
		require.NotEmpty(t, ev.GetResourceRetrieved().GetEventMetadata())
		require.NotEmpty(t, ev.GetResourceRetrieved().GetAuditContext())
		pbTest.CleanUpResourceRetrieved(ev.GetResourceRetrieved())
		require.NotEmpty(t, ev.GetResourceRetrieved().GetContent().GetData())
		ev.GetResourceRetrieved().GetContent().Data = nil
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
}

func checkAndValidateUpdate(ctx context.Context, t *testing.T, rac raservice.ResourceAggregateClient, s *subscription.Sub, recvChan <-chan *pb.Event, correlationID string, deviceID string, value uint64) {
	updCorrelationID := "updCorrelationID"
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

	check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_ResourceUpdatePending{
			ResourceUpdatePending: &events.ResourceUpdatePending{
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
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
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
						ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
						Content: &commands.Content{
							CoapContentFormat: -1,
						},
						Status:       commands.Status_OK,
						AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
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
						Status:       commands.Status_OK,
						AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
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
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
			},
		},
		CorrelationId: correlationID,
	})
	check(t, waitForEvent(ctx, t, recvChan), &pb.Event{
		SubscriptionId: s.Id(),
		Type: &pb.Event_ResourceRetrieved{
			ResourceRetrieved: &events.ResourceRetrieved{
				ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
				Content: &commands.Content{
					CoapContentFormat: int32(message.AppOcfCbor),
					ContentType:       message.AppOcfCbor.String(),
				},
				Status:       commands.Status_OK,
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
			},
		},
		CorrelationId: correlationID,
	})
}

func TestRequestHandlerSubscribeToEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	token := oauthTest.GetDefaultServiceToken(t)
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

	pool, err := ants.NewPool(1)
	require.NoError(t, err)
	natsConn, resourceSubscriber, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), log.Get(), subscriber.WithGoPool(pool.Submit), subscriber.WithUnmarshaler(utils.Unmarshal))
	require.NoError(t, err)
	defer natsConn.Close()
	defer resourceSubscriber.Close()

	ownerClaim := "sub"
	owner, err := kitNetGrpc.OwnerFromTokenMD(ctx, ownerClaim)
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
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				DeviceId: deviceID,
				Status: &commands.ConnectionStatus{
					Value: commands.ConnectionStatus_ONLINE,
				},
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
			},
		},
		CorrelationId: correlationID,
	})
	check(t, waitForEvent(ctx, t, recvChan), pbTest.ResourceLinkToPublishEvent(deviceID, correlationID, test.GetAllBackendResourceLinks()))

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
					Status:       commands.Status_OK,
					AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
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
