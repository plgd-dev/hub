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
	caService "github.com/plgd-dev/hub/v2/certificate-authority/test"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	grpcgwService "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	idService "github.com/plgd-dev/hub/v2/identity-store/test"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	rdTest "github.com/plgd-dev/hub/v2/resource-directory/test"
	"github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	serviceTest "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerSubscribeToEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := serviceTest.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resourceLinks := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.ACTIVE_COAP_SCHEME+testCfg.COAP_GW_HOST, resourceLinks)
	defer shutdownDevSim()

	const switchID = "1"
	resourceLinks = append(resourceLinks, test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)...)
	time.Sleep(200 * time.Millisecond)

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
				{
					Type:          pbTest.OperationProcessedOK(),
					CorrelationId: "testToken1",
				},
				{
					Type: &pb.Event_DeviceRegistered_{
						DeviceRegistered: &pb.Event_DeviceRegistered{
							DeviceIds: []string{deviceID},
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
				{
					Type:          pbTest.OperationProcessedOK(),
					CorrelationId: "testToken2",
				},
				{
					Type: &pb.Event_DeviceMetadataUpdated{
						DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
							DeviceId: deviceID,
							Status: &commands.ConnectionStatus{
								Value: commands.ConnectionStatus_ONLINE,
							},
							AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
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
				{
					Type:          pbTest.OperationProcessedOK(),
					CorrelationId: "testToken3",
				},
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
				assert.NoError(t, errC)
			}()
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				var events []*pb.Event
				for range tt.want {
					ev, err := client.Recv()
					if errors.Is(err, io.EOF) {
						break
					}
					require.NoError(t, err)
					events = append(events, ev)
				}
				pbTest.CmpEvents(t, tt.want, events)
			}()
			err = client.Send(tt.args.sub)
			require.NoError(t, err)
			wg.Wait()
		})
	}
}

func TestRequestHandlerSubscribeForCreateEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := serviceTest.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.ACTIVE_COAP_SCHEME+testCfg.COAP_GW_HOST, test.GetAllBackendResourceLinks())
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
	expectedEvent := &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type:           pbTest.OperationProcessedOK(),
		CorrelationId:  "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	const switchID = "1"
	switchData := test.MakeSwitchResourceDefaultData()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)

	ev, err = client.Recv()
	require.NoError(t, err)
	pbTest.CmpEvent(t, &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		CorrelationId:  "testToken",
		Type: &pb.Event_ResourceCreatePending{
			ResourceCreatePending: pbTest.MakeResourceCreatePending(t, deviceID, test.TestResourceSwitchesHref, "",
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
		SubscriptionId: ev.SubscriptionId,
		CorrelationId:  "testToken",
		Type: &pb.Event_ResourceCreated{
			ResourceCreated: pbTest.MakeResourceCreated(t, deviceID, test.TestResourceSwitchesHref, "", switchData),
		},
	}, ev, "")
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
					ResourceIdFilter: []string{
						commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")).ToString(),
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
						ResourceUpdatePending: pbTest.MakeResourceUpdatePending(t, deviceID, test.TestResourceLightInstanceHref("1"), "",
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
							UpdatePending: &events.DeviceMetadataUpdatePending_ShadowSynchronization{
								ShadowSynchronization: commands.ShadowSynchronization_DISABLED,
							},
							AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
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
							AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
						},
					},
				},
				{
					Command: &pb.PendingCommand_ResourceCreatePending{
						ResourceCreatePending: pbTest.MakeResourceCreatePending(t, deviceID, device.ResourceURI, "",
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
							AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
						},
					},
				},
				{
					Command: &pb.PendingCommand_ResourceUpdatePending{
						ResourceUpdatePending: pbTest.MakeResourceUpdatePending(t, deviceID, test.TestResourceLightInstanceHref("1"), "",
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
							AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
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
						ResourceCreatePending: pbTest.MakeResourceCreatePending(t, deviceID, device.ResourceURI, "",
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
							AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
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
						ResourceUpdatePending: pbTest.MakeResourceUpdatePending(t, deviceID, test.TestResourceLightInstanceHref("1"), "",
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
							UpdatePending: &events.DeviceMetadataUpdatePending_ShadowSynchronization{
								ShadowSynchronization: commands.ShadowSynchronization_DISABLED,
							},
							AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	serviceTest.ClearDB(ctx, t)
	oauthShutdown := oauthTest.SetUp(t)
	authShutdown := idService.SetUp(t)
	raShutdown := raService.SetUp(t)
	rdShutdown := rdTest.SetUp(t)
	grpcShutdown := grpcgwService.SetUp(t)
	caShutdown := caService.SetUp(t)
	secureGWShutdown := coapgwTest.SetUp(t)

	defer caShutdown()
	defer grpcShutdown()
	defer rdShutdown()
	defer raShutdown()
	defer authShutdown()
	defer oauthShutdown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	client, err := client.New(c).SubscribeToEventsWithCurrentState(ctx, time.Minute)
	require.NoError(t, err)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.ACTIVE_COAP_SCHEME+testCfg.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	secureGWShutdown()

	create := func(timeToLive time.Duration) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.CreateResource(ctx, &pb.CreateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
			TimeToLive: int64(timeToLive),
		})
		require.Error(t, err)
	}
	create(time.Millisecond * 500) // for test expired event
	create(0)

	retrieve := func(timeToLive time.Duration) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.GetResourceFromDevice(ctx, &pb.GetResourceFromDeviceRequest{
			ResourceId: commands.NewResourceID(deviceID, platform.ResourceURI),
			TimeToLive: int64(timeToLive),
		})
		require.Error(t, err)
	}
	retrieve(time.Millisecond * 500) // for test expired event
	retrieve(0)

	update := func(timeToLive time.Duration) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.UpdateResource(ctx, &pb.UpdateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
			TimeToLive: int64(timeToLive),
		})
		require.Error(t, err)
	}
	update(time.Millisecond * 500) // for test expired event
	update(0)

	delete := func(timeToLive time.Duration) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.DeleteResource(ctx, &pb.DeleteResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
			TimeToLive: int64(timeToLive),
		})
		require.Error(t, err)
	}
	delete(time.Millisecond * 500) // for test expired event
	delete(0)

	updateDeviceMetadata := func(timeToLive time.Duration) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
			DeviceId:              deviceID,
			ShadowSynchronization: pb.UpdateDeviceMetadataRequest_DISABLED,
			TimeToLive:            int64(timeToLive),
		})
		require.Error(t, err)
	}
	updateDeviceMetadata(time.Millisecond * 500) // for test expired event
	updateDeviceMetadata(0)                      // for test expired event

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
			expectedEvent := &pb.Event{
				SubscriptionId: ev.SubscriptionId,
				Type:           pbTest.OperationProcessedOK(),
				CorrelationId:  correlationID,
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
			subscriptionID := ev.SubscriptionId
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
			expectedEvent = &pb.Event{
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
			expectedEvent = &pb.Event{
				SubscriptionId: subscriptionID,
				Type:           pbTest.OperationProcessedOK(),
				CorrelationId:  correlationID,
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
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

	tearDown := serviceTest.SetUp(ctx, t, serviceTest.WithCOAPGWConfig(coapgwCfg), serviceTest.WithRDConfig(rdCfg), serviceTest.WithGRPCGWConfig(grpcgwCfg))
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	rdConn, err := grpcClient.New(testCfg.MakeGrpcClientConfig(testCfg.GRPC_GW_HOST), fileWatcher, log.Get(), trace.NewNoopTracerProvider())
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
	expectedEvent := &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type:           pbTest.OperationProcessedOK(),
		CorrelationId:  "testToken",
	}
	fmt.Printf("SUBSCRIPTION ID: %v\n", ev.SubscriptionId)
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type: &pb.Event_DeviceRegistered_{
			DeviceRegistered: &pb.Event_DeviceRegistered{
				DeviceIds: []string{},
			},
		},
		CorrelationId: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.ACTIVE_COAP_SCHEME+testCfg.COAP_GW_HOST, test.GetAllBackendResourceLinks())

	time.Sleep(time.Second * 10)

	ev, err = client.Recv()
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

	ev, err = client.Recv()
	require.NoError(t, err)
	pbTest.CmpEvent(t, &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				DeviceId: deviceID,
				Status: &commands.ConnectionStatus{
					Value: commands.ConnectionStatus_ONLINE,
				},
				AuditContext: commands.NewAuditContext(service.DeviceUserID, ""),
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
