package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	authService "github.com/plgd-dev/cloud/authorization/test"
	caService "github.com/plgd-dev/cloud/certificate-authority/test"
	coapgwTest "github.com/plgd-dev/cloud/coap-gateway/test"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	grpcgwService "github.com/plgd-dev/cloud/grpc-gateway/test"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	raService "github.com/plgd-dev/cloud/resource-aggregate/test"
	rdService "github.com/plgd-dev/cloud/resource-directory/test"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message"
)

func TestRequestHandler_SubscribeForPendingCommands(t *testing.T) {
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
			name: "without currentState",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATE_PENDING,
					},
				},
			},
			want: []*pb.PendingCommand{},
		},
		{
			name: "retrieve by resourceIdFilter",
			args: args{
				req: &pb.SubscribeToEvents_CreateSubscription{
					ResourceIdFilter: []string{
						commands.NewResourceID(deviceID, "/light/1").ToString(),
					},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_RETRIEVE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETE_PENDING,
						pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATE_PENDING,
					},
					IncludeCurrentState: true,
				},
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_ResourceUpdatePending{
						ResourceUpdatePending: &events.ResourceUpdatePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     "/light/1",
							},
							Content: &commands.Content{
								ContentType:       message.AppOcfCbor.String(),
								CoapContentFormat: -1,
								Data: test.EncodeToCbor(t, map[string]interface{}{
									"power": 1,
								}),
							},
							AuditContext: commands.NewAuditContext("1", ""),
						},
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
					IncludeCurrentState: true,
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
							AuditContext: commands.NewAuditContext("1", ""),
						},
					},
				},
				{
					Command: &pb.PendingCommand_ResourceRetrievePending{
						ResourceRetrievePending: &events.ResourceRetrievePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     "/oic/p",
							},
							AuditContext: commands.NewAuditContext("1", ""),
						},
					},
				},
				{
					Command: &pb.PendingCommand_ResourceCreatePending{
						ResourceCreatePending: &events.ResourceCreatePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     "/oic/d",
							},
							Content: &commands.Content{
								ContentType:       message.AppOcfCbor.String(),
								CoapContentFormat: -1,
								Data: test.EncodeToCbor(t, map[string]interface{}{
									"power": 1,
								}),
							},
							AuditContext: commands.NewAuditContext("1", ""),
						},
					},
				},
				{
					Command: &pb.PendingCommand_ResourceDeletePending{
						ResourceDeletePending: &events.ResourceDeletePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     "/oic/d",
							},
							AuditContext: commands.NewAuditContext("1", ""),
						},
					},
				},
				{
					Command: &pb.PendingCommand_ResourceUpdatePending{
						ResourceUpdatePending: &events.ResourceUpdatePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     "/light/1",
							},
							Content: &commands.Content{
								ContentType:       message.AppOcfCbor.String(),
								CoapContentFormat: -1,
								Data: test.EncodeToCbor(t, map[string]interface{}{
									"power": 1,
								}),
							},
							AuditContext: commands.NewAuditContext("1", ""),
						},
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
					IncludeCurrentState: true,
				},
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_ResourceRetrievePending{
						ResourceRetrievePending: &events.ResourceRetrievePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     "/oic/p",
							},
							AuditContext: commands.NewAuditContext("1", ""),
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
					IncludeCurrentState: true,
				},
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_ResourceCreatePending{
						ResourceCreatePending: &events.ResourceCreatePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     "/oic/d",
							},
							Content: &commands.Content{
								ContentType:       message.AppOcfCbor.String(),
								CoapContentFormat: -1,
								Data: test.EncodeToCbor(t, map[string]interface{}{
									"power": 1,
								}),
							},
							AuditContext: commands.NewAuditContext("1", ""),
						},
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
					IncludeCurrentState: true,
				},
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_ResourceDeletePending{
						ResourceDeletePending: &events.ResourceDeletePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     "/oic/d",
							},
							AuditContext: commands.NewAuditContext("1", ""),
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
					IncludeCurrentState: true,
				},
			},
			want: []*pb.PendingCommand{
				{
					Command: &pb.PendingCommand_ResourceUpdatePending{
						ResourceUpdatePending: &events.ResourceUpdatePending{
							ResourceId: &commands.ResourceId{
								DeviceId: deviceID,
								Href:     "/light/1",
							},
							Content: &commands.Content{
								ContentType:       message.AppOcfCbor.String(),
								CoapContentFormat: -1,
								Data: test.EncodeToCbor(t, map[string]interface{}{
									"power": 1,
								}),
							},
							AuditContext: commands.NewAuditContext("1", ""),
						},
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
					IncludeCurrentState: true,
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
							AuditContext: commands.NewAuditContext("1", ""),
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()

	test.ClearDB(ctx, t)
	oauthShutdown := oauthTest.SetUp(t)
	authShutdown := authService.SetUp(t)
	raShutdown := raService.SetUp(t)
	rdShutdown := rdService.SetUp(t)
	grpcShutdown := grpcgwService.SetUp(t)
	caShutdown := caService.SetUp(t)
	secureGWShutdown := coapgwTest.SetUp(t)

	defer caShutdown()
	defer grpcShutdown()
	defer rdShutdown()
	defer raShutdown()
	defer authShutdown()
	defer oauthShutdown()

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	client, err := c.SubscribeToEvents(ctx)
	require.NoError(t, err)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	secureGWShutdown()

	create := func(timeToLive time.Duration) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.CreateResource(ctx, &pb.CreateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
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
			ResourceId: commands.NewResourceID(deviceID, "/oic/p"),
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
			ResourceId: commands.NewResourceID(deviceID, "/light/1"),
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
			ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("test %v\n", tt.name)
			// register for subscription
			err = client.Send(&pb.SubscribeToEvents{
				CorrelationId: "testToken",
				Action: &pb.SubscribeToEvents_CreateSubscription_{
					CreateSubscription: tt.args.req,
				},
			})
			require.NoError(t, err)

			ev, err := client.Recv()
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
			subscriptionId := ev.SubscriptionId
			fmt.Printf("sub %v\n", subscriptionId)

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
			cmpPendingCmds(t, tt.want, values)
			err = client.Send(&pb.SubscribeToEvents{
				CorrelationId: "testToken",
				Action: &pb.SubscribeToEvents_CancelSubscription_{
					CancelSubscription: &pb.SubscribeToEvents_CancelSubscription{
						SubscriptionId: subscriptionId,
					},
				},
			})
			require.NoError(t, err)

			// cancellation event
			ev, err = client.Recv()
			require.NoError(t, err)
			expectedEvent = &pb.Event{
				SubscriptionId: subscriptionId,
				Type: &pb.Event_SubscriptionCanceled_{
					SubscriptionCanceled: &pb.Event_SubscriptionCanceled{},
				},
				CorrelationId: "testToken",
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

			// response for close subscription
			ev, err = client.Recv()
			require.NoError(t, err)
			expectedEvent = &pb.Event{
				SubscriptionId: subscriptionId,
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
		})
	}
}
