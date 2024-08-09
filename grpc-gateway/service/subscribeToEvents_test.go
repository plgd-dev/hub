package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/go-coap/v3/message"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/service"
	grpcgwService "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	rdTest "github.com/plgd-dev/hub/v2/resource-directory/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerSubscribeToEvents(t *testing.T) {
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

	resourceLinks := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, resourceLinks)
	defer shutdownDevSim()

	const switchID = "1"
	resourceLinks = append(resourceLinks, test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)...)
	// for update resource-directory cache
	time.Sleep(time.Second)

	type args struct {
		sub *pb.SubscribeToEvents
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
					CorrelationId: "testToken0",
				},
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
					CorrelationId: "testToken0",
				},
			},
		},
		{
			name: "devices subscription - registered",
			args: args{
				sub: &pb.SubscribeToEvents{
					CorrelationId: "testToken1",
					Action: &pb.SubscribeToEvents_CreateSubscription_{
						CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
							EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
								pb.SubscribeToEvents_CreateSubscription_REGISTERED,
								pb.SubscribeToEvents_CreateSubscription_UNREGISTERED,
							},
						},
					},
				},
			},
			want: []*pb.Event{
				pbTest.NewOperationProcessedOK("", "testToken1"),
				{
					Type: &pb.Event_DeviceRegistered_{
						DeviceRegistered: &pb.Event_DeviceRegistered{
							DeviceIds: []string{deviceID},
							EventMetadata: &isEvents.EventMetadata{
								HubId: config.HubID(),
							},
						},
					},
					CorrelationId: "testToken1",
				},
			},
		},
		{
			name: "devices subscription - online",
			args: args{
				sub: &pb.SubscribeToEvents{
					CorrelationId: "testToken2",
					Action: &pb.SubscribeToEvents_CreateSubscription_{
						CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
							EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
								pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
							},
						},
					},
				},
			},
			want: []*pb.Event{
				pbTest.NewOperationProcessedOK("", "testToken2"),
				{
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
					CorrelationId: "testToken2",
				},
			},
		},
		{
			name: "device subscription - published",
			args: args{
				sub: &pb.SubscribeToEvents{
					CorrelationId: "testToken3",
					Action: &pb.SubscribeToEvents_CreateSubscription_{
						CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
							DeviceIdFilter: []string{deviceID},
							EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
								pb.SubscribeToEvents_CreateSubscription_RESOURCE_PUBLISHED,
								pb.SubscribeToEvents_CreateSubscription_RESOURCE_UNPUBLISHED,
							},
						},
					},
				},
			},
			want: []*pb.Event{
				pbTest.NewOperationProcessedOK("", "testToken3"),
				pbTest.ResourceLinkToPublishEvent(deviceID, "testToken3", resourceLinks),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := client.New(c).SubscribeToEventsWithCurrentState(ctx, time.Minute)
			require.NoError(t, err)
			defer func() {
				errC := client.CloseSend()
				require.NoError(t, errC)
			}()
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				var events []*pb.Event
				for range tt.want {
					ev, errR := client.Recv()
					if errors.Is(errR, io.EOF) {
						break
					}
					assert.NoError(t, errR)
					events = append(events, ev)
				}
				pbTest.AssertCmpEvents(t, tt.want, events)
			}()
			err = client.Send(tt.args.sub)
			require.NoError(t, err)
			wg.Wait()
		})
	}
}

func TestRequestHandlerSubscribeForCreateEvents(t *testing.T) {
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

	client, err := client.New(c).SubscribeToEventsWithCurrentState(ctx, time.Minute)
	require.NoError(t, err)

	err = client.Send(&pb.SubscribeToEvents{
		CorrelationId: "testToken",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATE_PENDING,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATED,
				},
			},
		},
	})
	require.NoError(t, err)

	ev, err := client.Recv()
	require.NoError(t, err)
	test.CheckProtobufs(t, pbTest.NewOperationProcessedOK(ev.GetSubscriptionId(), "testToken"), ev, test.RequireToCheckFunc(require.Equal))
	subID := ev.GetSubscriptionId()

	const switchID = "1"
	switchData := test.MakeSwitchResourceDefaultData()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)

	ev, err = client.Recv()
	require.NoError(t, err)
	pbTest.CmpEvent(t, &pb.Event{
		SubscriptionId: ev.GetSubscriptionId(),
		CorrelationId:  "testToken",
		Type: &pb.Event_ResourceCreatePending{
			ResourceCreatePending: pbTest.MakeResourceCreatePending(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, "",
				switchData),
		},
	}, ev, "")

	switchLink := test.DefaultSwitchResourceLink("", switchID)
	switchData = test.MakeSwitchResourceData(map[string]interface{}{
		"href": switchLink.Href,
		"rep": map[string]interface{}{
			"if":    switchLink.Interfaces,
			"rt":    switchLink.ResourceTypes,
			"value": false,
		},
	})

	ev, err = client.Recv()
	require.NoError(t, err)
	pbTest.CmpEvent(t, &pb.Event{
		SubscriptionId: ev.GetSubscriptionId(),
		CorrelationId:  "testToken",
		Type: &pb.Event_ResourceCreated{
			ResourceCreated: pbTest.MakeResourceCreated(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, "", switchData),
		},
	}, ev, "")

	// fail to cancel subscription - invalid subscriptionId
	err = client.Send(&pb.SubscribeToEvents{
		CorrelationId: "failCancel",
		Action: &pb.SubscribeToEvents_CancelSubscription_{
			CancelSubscription: &pb.SubscribeToEvents_CancelSubscription{
				SubscriptionId: "invalid",
			},
		},
	})
	require.NoError(t, err)

	ev, err = client.Recv()
	require.NoError(t, err)
	pbTest.CmpEvent(t, service.NewOperationProcessed(ev.GetSubscriptionId(), "failCancel", pb.Event_OperationProcessed_ErrorStatus_NOT_FOUND,
		ev.GetOperationProcessed().GetErrorStatus().GetMessage()), ev, "")

	// cancel subscription
	err = client.Send(&pb.SubscribeToEvents{
		CorrelationId: "cancel",
		Action: &pb.SubscribeToEvents_CancelSubscription_{
			CancelSubscription: &pb.SubscribeToEvents_CancelSubscription{
				SubscriptionId: subID,
			},
		},
	})
	require.NoError(t, err)

	ev, err = client.Recv()
	require.NoError(t, err)
	pbTest.CmpEvent(t, &pb.Event{
		SubscriptionId: subID,
		Type: &pb.Event_SubscriptionCanceled_{
			SubscriptionCanceled: &pb.Event_SubscriptionCanceled{},
		},
		CorrelationId: "cancel",
	}, ev, "")

	ev, err = client.Recv()
	require.NoError(t, err)
	pbTest.CmpEvent(t, pbTest.NewOperationProcessedOK(subID, "cancel"), ev, "")
}

func TestRequestHandlerSubscribeForHrefEvents(t *testing.T) {
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

	client, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)

	err = client.Send(&pb.SubscribeToEvents{
		CorrelationId: "testToken",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				HrefFilter: []string{
					test.TestResourceSwitchesHref,
				},
			},
		},
	})
	require.NoError(t, err)

	ev, err := client.Recv()
	require.NoError(t, err)
	test.CheckProtobufs(t, pbTest.NewOperationProcessedOK(ev.GetSubscriptionId(), "testToken"), ev, test.RequireToCheckFunc(require.Equal))

	const switchID = "1"
	switchData := test.MakeSwitchResourceDefaultData()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)

	for {
		ev, err = client.Recv()
		require.NoError(t, err)
		if v, ok := ev.GetType().(interface{ GetResourceId() *commands.ResourceId }); ok {
			require.Equal(t, test.TestResourceSwitchesHref, v.GetResourceId().GetHref())
		}
		if ev.GetResourceCreatePending() != nil {
			pbTest.CmpEvent(t, &pb.Event{
				SubscriptionId: ev.GetSubscriptionId(),
				CorrelationId:  "testToken",
				Type: &pb.Event_ResourceCreatePending{
					ResourceCreatePending: pbTest.MakeResourceCreatePending(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, "",
						switchData),
				},
			}, ev, "")
			break
		}
	}

	time.Sleep(time.Second)

	switchLink := test.DefaultSwitchResourceLink("", switchID)
	switchData = test.MakeSwitchResourceData(map[string]interface{}{
		"href": switchLink.Href,
		"rep": map[string]interface{}{
			"if":    switchLink.Interfaces,
			"rt":    switchLink.ResourceTypes,
			"value": false,
		},
	})

	for {
		ev, err = client.Recv()
		require.NoError(t, err)
		if v, ok := ev.GetType().(interface{ GetResourceId() *commands.ResourceId }); ok {
			require.Equal(t, test.TestResourceSwitchesHref, v.GetResourceId().GetHref())
		}
		if ev.GetResourceCreated() != nil {
			pbTest.CmpEvent(t, &pb.Event{
				SubscriptionId: ev.GetSubscriptionId(),
				CorrelationId:  "testToken",
				Type: &pb.Event_ResourceCreated{
					ResourceCreated: pbTest.MakeResourceCreated(t, deviceID, test.TestResourceSwitchesHref, test.TestResourceSwitchesResourceTypes, "", switchData),
				},
			}, ev, "")
			break
		}
	}
}

func TestRequestHandlerSubscribeForPendingCommands(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.SubscribeToEvents_CreateSubscription
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.PendingCommand
	}{
		{
			name: "retrieve by resourceIdFilter",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					ResourceIdFilter: []*pb.ResourceIdFilter{
						{
							ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
						},
					},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATE_PENDING,
					},
				},
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_ResourceUpdatePending{
						ResourceUpdatePending: pbTest.MakeResourceUpdatePending(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "",
							map[string]interface{}{
								"power": 1,
							}),
					},
				},
			},
		},
		{
			name: "retrieve by deviceIdFilter",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					DeviceIdFilter: []string{deviceID},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATE_PENDING,
					},
				},
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_DeviceMetadataUpdatePending{
						DeviceMetadataUpdatePending: &events.DeviceMetadataUpdatePending{
							DeviceId: deviceID,
							UpdatePending: &events.DeviceMetadataUpdatePending_TwinEnabled{
								TwinEnabled: false,
							},
							AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
						},
					},
				},
				{
					Command: &pb.PendingCommand_ResourceRetrievePending{
						ResourceRetrievePending: &events.ResourceRetrievePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     platform.ResourceURI,
							},
							AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
							ResourceTypes: []string{platform.ResourceType},
						},
					},
				},
				{
					Command: &pb.PendingCommand_ResourceCreatePending{
						ResourceCreatePending: pbTest.MakeResourceCreatePending(t, deviceID, device.ResourceURI, test.TestResourceDeviceResourceTypes, "",
							map[string]interface{}{
								"power": 1,
							}),
					},
				},
				{
					Command: &pb.PendingCommand_ResourceDeletePending{
						ResourceDeletePending: &events.ResourceDeletePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     device.ResourceURI,
							},
							AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
							ResourceTypes: test.TestResourceDeviceResourceTypes,
						},
					},
				},
				{
					Command: &pb.PendingCommand_ResourceUpdatePending{
						ResourceUpdatePending: pbTest.MakeResourceUpdatePending(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "",
							map[string]interface{}{
								"power": 1,
							}),
					},
				},
			},
		},
		{
			name: "filter retrieve commands",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING,
					},
				},
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_ResourceRetrievePending{
						ResourceRetrievePending: &events.ResourceRetrievePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     platform.ResourceURI,
							},
							AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
							ResourceTypes: []string{platform.ResourceType},
						},
					},
				},
			},
		},
		{
			name: "filter create commands",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATE_PENDING,
					},
				},
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_ResourceCreatePending{
						ResourceCreatePending: pbTest.MakeResourceCreatePending(t, deviceID, device.ResourceURI, test.TestResourceDeviceResourceTypes, "",
							map[string]interface{}{
								"power": 1,
							}),
					},
				},
			},
		},
		{
			name: "filter delete commands",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					DeviceIdFilter: []string{deviceID},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETE_PENDING,
					},
				},
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_ResourceDeletePending{
						ResourceDeletePending: &events.ResourceDeletePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     device.ResourceURI,
							},
							AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
							ResourceTypes: test.TestResourceDeviceResourceTypes,
						},
					},
				},
			},
		},
		{
			name: "filter update commands",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING,
					},
				},
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_ResourceUpdatePending{
						ResourceUpdatePending: pbTest.MakeResourceUpdatePending(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "",
							map[string]interface{}{
								"power": 1,
							}),
					},
				},
			},
		},
		{
			name: "filter device metadata update",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATE_PENDING,
					},
				},
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_DeviceMetadataUpdatePending{
						DeviceMetadataUpdatePending: &events.DeviceMetadataUpdatePending{
							DeviceId: deviceID,
							UpdatePending: &events.DeviceMetadataUpdatePending_TwinEnabled{
								TwinEnabled: false,
							},
							AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	hubTestService.ClearDB(ctx, t)

	const services = hubTestService.SetUpServicesMachine2MachineOAuth | hubTestService.SetUpServicesOAuth | hubTestService.SetUpServicesId | hubTestService.SetUpServicesResourceAggregate |
		hubTestService.SetUpServicesResourceDirectory | hubTestService.SetUpServicesCertificateAuthority | hubTestService.SetUpServicesGrpcGateway
	tearDown := hubTestService.SetUpServices(ctx, t, services)
	defer tearDown()

	deferedSecureGWShutdown := true
	secureGWShutdown := coapgwTest.SetUp(t)
	defer func() {
		if deferedSecureGWShutdown {
			secureGWShutdown()
		}
	}()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	client, err := client.New(c).SubscribeToEventsWithCurrentState(ctx, time.Minute)
	require.NoError(t, err)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	deferedSecureGWShutdown = false
	secureGWShutdown()

	createFn := func(timeToLive time.Duration) {
		_, errC := c.CreateResource(ctx, &pb.CreateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
			TimeToLive: int64(timeToLive),
			Async:      true,
		})
		require.NoError(t, errC)
	}
	createFn(time.Millisecond * 500) // for test expired event
	createFn(0)

	retrieveFn := func(timeToLive time.Duration) {
		retrieveCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, errG := c.GetResourceFromDevice(retrieveCtx, &pb.GetResourceFromDeviceRequest{
			ResourceId: commands.NewResourceID(deviceID, platform.ResourceURI),
			TimeToLive: int64(timeToLive),
		})
		require.Error(t, errG)
	}
	retrieveFn(time.Millisecond * 500) // for test expired event
	retrieveFn(0)

	updateFn := func(timeToLive time.Duration) {
		_, errU := c.UpdateResource(ctx, &pb.UpdateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
			TimeToLive: int64(timeToLive),
			Async:      true,
		})
		require.NoError(t, errU)
	}
	updateFn(time.Millisecond * 500) // for test expired event
	updateFn(0)

	deleteFn := func(timeToLive time.Duration) {
		_, errD := c.DeleteResource(ctx, &pb.DeleteResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
			TimeToLive: int64(timeToLive),
			Async:      true,
		})
		require.NoError(t, errD)
	}
	deleteFn(time.Millisecond * 500) // for test expired event
	deleteFn(0)

	updateDeviceMetadataFn := func(timeToLive time.Duration) {
		updateCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, errU := c.UpdateDeviceMetadata(updateCtx, &pb.UpdateDeviceMetadataRequest{
			DeviceId:    deviceID,
			TwinEnabled: false,
			TimeToLive:  int64(timeToLive),
		})
		require.Error(t, errU)
	}
	updateDeviceMetadataFn(time.Millisecond * 500) // for test expired event
	updateDeviceMetadataFn(0)                      // for test expired event

	for idx, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("test %v\n", tt.name)
			// register for subscription
			correlationID := fmt.Sprintf("testToken%v", idx)
			err = client.Send(&pb.SubscribeToEvents{
				CorrelationId: correlationID,
				Action: &pb.SubscribeToEvents_CreateSubscription_{
					CreateSubscription: tt.args.req,
				},
			})
			require.NoError(t, err)

			ev, err := client.Recv()
			require.NoError(t, err)
			test.CheckProtobufs(t, pbTest.NewOperationProcessedOK(ev.GetSubscriptionId(), correlationID), ev, test.RequireToCheckFunc(require.Equal))
			subscriptionID := ev.GetSubscriptionId()
			fmt.Printf("sub %v\n", subscriptionID)

			values := make([]*pb.PendingCommand, 0, 1)
			for len(values) < len(tt.want) {
				ev, err = client.Recv()
				require.NoError(t, err)
				fmt.Printf("ev %+v\n", ev)
				switch {
				case ev.GetResourceCreatePending() != nil:
					values = append(values, &pb.PendingCommand{Command: &pb.PendingCommand_ResourceCreatePending{ResourceCreatePending: ev.GetResourceCreatePending()}})
				case ev.GetResourceRetrievePending() != nil:
					values = append(values, &pb.PendingCommand{Command: &pb.PendingCommand_ResourceRetrievePending{ResourceRetrievePending: ev.GetResourceRetrievePending()}})
				case ev.GetResourceUpdatePending() != nil:
					values = append(values, &pb.PendingCommand{Command: &pb.PendingCommand_ResourceUpdatePending{ResourceUpdatePending: ev.GetResourceUpdatePending()}})
				case ev.GetResourceDeletePending() != nil:
					values = append(values, &pb.PendingCommand{Command: &pb.PendingCommand_ResourceDeletePending{ResourceDeletePending: ev.GetResourceDeletePending()}})
				case ev.GetDeviceMetadataUpdatePending() != nil:
					values = append(values, &pb.PendingCommand{Command: &pb.PendingCommand_DeviceMetadataUpdatePending{DeviceMetadataUpdatePending: ev.GetDeviceMetadataUpdatePending()}})
				}
			}
			pbTest.CmpPendingCmds(t, tt.want, values)
			err = client.Send(&pb.SubscribeToEvents{
				CorrelationId: correlationID,
				Action: &pb.SubscribeToEvents_CancelSubscription_{
					CancelSubscription: &pb.SubscribeToEvents_CancelSubscription{
						SubscriptionId: subscriptionID,
					},
				},
			})
			require.NoError(t, err)

			// cancellation event
			ev, err = client.Recv()
			require.NoError(t, err)
			expectedEvent := &pb.Event{
				SubscriptionId: subscriptionID,
				Type: &pb.Event_SubscriptionCanceled_{
					SubscriptionCanceled: &pb.Event_SubscriptionCanceled{},
				},
				CorrelationId: correlationID,
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

			// response for close subscription
			ev, err = client.Recv()
			require.NoError(t, err)
			test.CheckProtobufs(t, pbTest.NewOperationProcessedOK(subscriptionID, correlationID), ev, test.RequireToCheckFunc(require.Equal))
		})
	}
}

func TestRequestHandlerIssue270(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	coapgwCfg := coapgwTest.MakeConfig(t)
	rdCfg := rdTest.MakeConfig(t)
	grpcgwCfg := grpcgwService.MakeConfig(t)

	tearDown := hubTestService.SetUp(ctx, t, hubTestService.WithCOAPGWConfig(coapgwCfg), hubTestService.WithRDConfig(rdCfg), hubTestService.WithGRPCGWConfig(grpcgwCfg))
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	rdConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.GRPC_GW_HOST), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = rdConn.Close()
	}()
	c := pb.NewGrpcGatewayClient(rdConn.GRPC())

	client, err := client.New(c).SubscribeToEventsWithCurrentState(ctx, time.Minute)
	require.NoError(t, err)

	err = client.Send(&pb.SubscribeToEvents{
		CorrelationId: "testToken",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
					pb.SubscribeToEvents_CreateSubscription_REGISTERED,
					pb.SubscribeToEvents_CreateSubscription_UNREGISTERED,
				},
			},
		},
	})
	require.NoError(t, err)

	ev, err := client.Recv()
	require.NoError(t, err)
	fmt.Printf("SUBSCRIPTION ID: %v\n", ev.GetSubscriptionId())
	test.CheckProtobufs(t, pbTest.NewOperationProcessedOK(ev.GetSubscriptionId(), "testToken"), ev, test.RequireToCheckFunc(require.Equal))

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent := &pb.Event{
		SubscriptionId: ev.GetSubscriptionId(),
		Type: &pb.Event_DeviceRegistered_{
			DeviceRegistered: &pb.Event_DeviceRegistered{
				DeviceIds: []string{},
			},
		},
		CorrelationId: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())

	time.Sleep(time.Second * 10)

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.GetSubscriptionId(),
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
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	ev, err = client.Recv()
	require.NoError(t, err)
	pbTest.CmpEvent(t, &pb.Event{
		SubscriptionId: ev.GetSubscriptionId(),
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
		CorrelationId: "testToken",
	}, ev, "")

	shutdownDevSim()
	run := true
	for run {
		ev, err = client.Recv()
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
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
			run = false
		}
	}
}

func waitForDevice(t *testing.T, client pb.GrpcGateway_SubscribeToEventsClient, deviceID string) string {
	// device should be online
	var firstCoapGWInstanceID string
	for {
		ev, err := client.Recv()
		require.NoError(t, err)
		if firstCoapGWInstanceID == "" {
			firstCoapGWInstanceID = ev.GetDeviceMetadataUpdated().GetConnection().GetServiceId()
		}
		require.Equal(t, firstCoapGWInstanceID, ev.GetDeviceMetadataUpdated().GetConnection().GetServiceId())
		wantBreak := ev.GetDeviceMetadataUpdated().GetTwinSynchronization().GetState() == commands.TwinSynchronization_IN_SYNC
		// this alternate to multiple values
		ev.GetDeviceMetadataUpdated().TwinSynchronization = nil
		pbTest.CmpEvent(t, &pb.Event{
			SubscriptionId: ev.GetSubscriptionId(),
			Type: &pb.Event_DeviceMetadataUpdated{
				DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
					DeviceId: deviceID,
					Connection: &commands.Connection{
						Status:   commands.Connection_ONLINE,
						Protocol: test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
					},
					TwinEnabled:  true,
					AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
				},
			},
			CorrelationId: "testToken",
		}, ev, "")
		if wantBreak {
			return firstCoapGWInstanceID
		}
	}
}

func TestCoAPGatewayServiceHeartbeat(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	// This test should be constrained to a 3-minute time limit.
	// The reason for this limit is that the device's DTLS system identifies
	// a connection loss within 20 seconds for the initial ping, and subsequently,
	// there are 5 more pings, each separated by 4 seconds, with a timeout of 4 seconds.
	// Therefore, the total time for this sequence is 20 + 4 + (5 * (4 + 4)) = 64 seconds,
	// plus an additional 10 seconds for waiting for the device to come online.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*3)
	defer cancel()

	tearDown := hubTestService.SetUpServices(ctx, t, hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesCertificateAuthority|hubTestService.SetUpServicesGrpcGateway|
		hubTestService.SetUpServicesId|hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesOAuth)
	defer tearDown()

	racfg := raTest.MakeConfig(t)
	raTearDown := raTest.New(t, racfg)

	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.ServiceHeartbeat.TimeToLive = time.Second * 3
	coapgwTearDown := coapgwTest.New(t, coapgwCfg)

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	rdConn, err := grpcClient.New(config.MakeGrpcClientConfig(config.GRPC_GW_HOST), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	defer func() {
		_ = rdConn.Close()
	}()
	c := pb.NewGrpcGatewayClient(rdConn.GRPC())

	client, err := client.New(c).SubscribeToEventsWithCurrentState(ctx, time.Minute)
	require.NoError(t, err)

	err = client.Send(&pb.SubscribeToEvents{
		CorrelationId: "testToken",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
				},
			},
		},
	})
	require.NoError(t, err)

	ev, err := client.Recv()
	require.NoError(t, err)
	test.CheckProtobufs(t, pbTest.NewOperationProcessedOK(ev.GetSubscriptionId(), "testToken"), ev, test.RequireToCheckFunc(require.Equal))

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	// wait for device to be online
	firstCoapGWInstanceID := waitForDevice(t, client, deviceID)
	require.NotEmpty(t, firstCoapGWInstanceID)

	// need to turn off RA to don't allow update device status to offline
	raTearDown()
	// turn off coapgw - devices status will still be online
	coapgwTearDown()
	time.Sleep(time.Second)

	// turn on resource-aggregate
	raTearDown = raTest.New(t, racfg)

	// turn on coapgw on different port to avoid connecting device to hub
	// in this case this coapgw will move device to offline
	coapgwCfg.APIs.COAP.Addr = "localhost:55555"
	coapgwTearDown = coapgwTest.New(t, coapgwCfg)

	ev, err = client.Recv()
	require.NoError(t, err)
	pbTest.CmpEvent(t, &pb.Event{
		SubscriptionId: ev.GetSubscriptionId(),
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				DeviceId: deviceID,
				Connection: &commands.Connection{
					Status:   commands.Connection_OFFLINE,
					Protocol: test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
				},
				TwinEnabled:         true,
				TwinSynchronization: &commands.TwinSynchronization{},
				AuditContext:        commands.NewAuditContext(raService.ServiceUserID, "", oauthService.DeviceUserID),
			},
		},
		CorrelationId: "testToken",
	}, ev, "")

	// turn off coap-gw and return back correct port
	coapgwTearDown()
	time.Sleep(time.Second)
	coapgwCfg.APIs.COAP.Addr = config.COAP_GW_HOST
	coapgwTearDown = coapgwTest.New(t, coapgwCfg)

	// device should be online again
	secondCoapGWInstanceID := waitForDevice(t, client, deviceID)
	require.NotEmpty(t, secondCoapGWInstanceID)
	require.NotEqual(t, firstCoapGWInstanceID, secondCoapGWInstanceID)

	// ---- Set the device to offline via the resource-aggregate without updating the service metadata through the CoAP gateway. ---
	// turn off resource-aggregate
	raTearDown()
	coapgwTearDown()
	time.Sleep(time.Second)

	// turn on resource-aggregate
	raTearDown = raTest.New(t, racfg)
	defer raTearDown()

	// device should go to offline
	ev, err = client.Recv()
	require.NoError(t, err)
	pbTest.CmpEvent(t, &pb.Event{
		SubscriptionId: ev.GetSubscriptionId(),
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				DeviceId: deviceID,
				Connection: &commands.Connection{
					Status:   commands.Connection_OFFLINE,
					Protocol: test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
				},
				TwinEnabled:         true,
				TwinSynchronization: &commands.TwinSynchronization{},
				AuditContext:        commands.NewAuditContext(raService.ServiceUserID, "", oauthService.DeviceUserID),
			},
		},
		CorrelationId: "testToken",
	}, ev, "")

	// turn on coap-gw
	coapgwTearDown = coapgwTest.New(t, coapgwCfg)
	defer coapgwTearDown()

	// device should be online again
	thirdCoapGWInstanceID := waitForDevice(t, client, deviceID)
	require.NotEmpty(t, thirdCoapGWInstanceID)
	require.NotEqual(t, firstCoapGWInstanceID, thirdCoapGWInstanceID)
}
