package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	coapTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	isPb "github.com/plgd-dev/hub/v2/identity-store/pb"
	isTest "github.com/plgd-dev/hub/v2/identity-store/test"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestUpdateResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	switchID := "1"
	type args struct {
		req *pb.UpdateResourceRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *events.ResourceUpdated
		wantErr bool
	}{
		{
			name: "invalid Href",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/unknown"),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid timeToLive",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
					TimeToLive: int64(99 * time.Millisecond),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 1,
						}),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid update RO-resource",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, device.ResourceURI),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"di": "abc",
						}),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid update collection /switches",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesHref),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"href": test.TestResourceSwitchesInstanceHref(switchID),
						}),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 1,
						}),
					},
				},
			},
			want: pbTest.MakeResourceUpdated(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "", nil),
		},
		{
			name: "valid with interface",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceInterface: interfaces.OC_IF_BASELINE,
					ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 2,
						}),
					},
				},
			},
			want: pbTest.MakeResourceUpdated(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "", nil),
		},
		{
			name: "revert update",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceInterface: interfaces.OC_IF_BASELINE,
					ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 0,
						}),
					},
				},
			},
			want: pbTest.MakeResourceUpdated(t, deviceID, test.TestResourceLightInstanceHref("1"), test.TestResourceLightInstanceResourceTypes, "", nil),
		},
		{
			name: "update /switches/1",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID)),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"value": true,
						}),
					},
				},
			},
			want: &events.ResourceUpdated{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     test.TestResourceSwitchesInstanceHref(switchID),
				},
				Content: &commands.Content{
					CoapContentFormat: int32(message.AppOcfCbor),
					ContentType:       message.AppOcfCbor.String(),
					Data: test.EncodeToCbor(t, map[string]interface{}{
						"value": true,
					}),
				},
				Status:        commands.Status_OK,
				AuditContext:  commands.NewAuditContext(oauthService.DeviceUserID, "", oauthService.DeviceUserID),
				ResourceTypes: test.TestResourceSwitchesInstanceResourceTypes,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
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

	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)
	// for update resource-directory cache
	time.Sleep(time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.UpdateResource(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			pbTest.CmpResourceUpdated(t, tt.want, got.GetData())
		})
	}
}

// Iotivity-lite - oic/d piid issue with notification (#40)
// https://github.com/iotivity/iotivity-lite/issues/40
//
// After updating the device name using /oc/con resource the piid
// field disappears from the /oic/d resource.
func TestRequestHandlerGetAfterUpdateResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
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

	getData := func(devID string) map[string]interface{} {
		resources := make(map[string]interface{})
		client, err := c.GetResources(ctx, &pb.GetResourcesRequest{
			DeviceIdFilter: []string{devID},
		})
		require.NoError(t, err)
		for {
			value, err := client.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
			resources[value.GetData().GetResourceId().ToString()] = test.DecodeCbor(t, value.GetData().GetContent().GetData())
		}
		return resources
	}

	startData := getData(deviceID)
	updateName := func(name string) {
		_, err := c.UpdateResource(ctx, &pb.UpdateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, configuration.ResourceURI),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"n": name,
				}),
			},
		})
		require.NoError(t, err)
		// for update resource-directory cache
		time.Sleep(time.Second)
	}
	updateName("update simulator")
	// revert name
	updateName(test.TestDeviceName)

	endData := getData(deviceID)
	require.Equal(t, startData, endData)
}

func TestRequestHandlerRunMultipleUpdateResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	numIteration := 200
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT+(time.Second*3*time.Duration(numIteration)))
	defer cancel()

	raCfg := raTest.MakeConfig(t)
	tearDown := service.SetUp(ctx, t, service.WithRAConfig(raCfg))
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

	resources := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, resources)
	defer shutdownDevSim()

	for i := range numIteration {
		func() {
			t.Logf("TestRequestHandlerMultipleUpdateResource:run %v\n", i)
			lightHref := test.TestResourceLightInstanceHref("1")
			subCtx, cancel := context.WithTimeout(ctx, time.Second*3)
			defer cancel()
			subClient, err := c.SubscribeToEvents(subCtx)
			require.NoError(t, err)
			defer func() {
				errC := subClient.CloseSend()
				require.NoError(t, errC)
			}()

			err = subClient.Send(&pb.SubscribeToEvents{Action: &pb.SubscribeToEvents_CreateSubscription_{
				CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED},
					ResourceIdFilter: []*pb.ResourceIdFilter{
						{
							ResourceId: commands.NewResourceID(deviceID, lightHref),
						},
					},
				},
			}})
			require.NoError(t, err)
			ev, err := subClient.Recv()
			require.NoError(t, err)
			require.Equal(t, pb.Event_OperationProcessed_ErrorStatus_OK, ev.GetOperationProcessed().GetErrorStatus().GetCode())
			for j := 1; j >= 0; j-- {
				_, err = c.UpdateResource(subCtx, &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, lightHref),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": j,
						}),
					},
				})
				require.NoError(t, err)
			}
			makeLightData := func(power int) map[string]interface{} {
				return map[string]interface{}{
					"name":  "Light",
					"power": uint64(power),
					"state": false,
				}
			}
			for j := 1; j >= 0; j-- {
				ev, err = subClient.Recv()
				require.NoError(t, err)
				pbTest.CmpResourceChanged(t, pbTest.MakeResourceChanged(t, deviceID, lightHref, test.TestResourceLightInstanceResourceTypes, "", makeLightData(j)), ev.GetResourceChanged(), "")
			}
		}()
	}
}

func TestRequestHandlerRunMultipleParallelUpdateResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	numIteration := 20
	timeout := time.Second * 120
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	raCfg := raTest.MakeConfig(t)
	raCfg.Clients.Eventstore.ConcurrencyExceptionMaxRetry = 40

	coapCfg := coapTest.MakeConfig(t)
	tearDown := service.SetUp(ctx, t, service.WithRAConfig(raCfg), service.WithCOAPGWConfig(coapCfg))
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

	resources := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, resources)
	defer shutdownDevSim()

	var wg sync.WaitGroup
	wg.Add(numIteration)
	for i := range numIteration {
		t.Logf("TestRequestHandlerRunMultipleParallelUpdateResource:run %v\n", i)
		go func() {
			defer wg.Done()
			lightHref := test.TestResourceLightInstanceHref("1")
			updateCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			for j := 1; j >= 0; j-- {
				_, err := c.UpdateResource(updateCtx, &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, lightHref),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": j,
						}),
					},
				})
				assert.NoError(t, err)
			}
		}()
	}
	wg.Wait()
}

func TestUpdateCreateOnNotExistingResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	switchID := "1"

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	isCfg := isTest.MakeConfig(t)
	isCfg.APIs.GRPC.TLS.ClientCertificateRequired = false
	tearDown := service.SetUp(ctx, t, service.WithISConfig(isCfg))
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	// associate device with owner
	isConn, err := grpc.NewClient(config.IDENTITY_STORE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = isConn.Close()
	}()
	isClient := isPb.NewIdentityStoreClient(isConn)
	_, err = isClient.AddDevice(ctx, &isPb.AddDeviceRequest{
		DeviceId: deviceID,
	})
	require.NoError(t, err)

	// update/create resources of the registered device
	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	createResourceSub := &pb.SubscribeToEvents{
		CorrelationId: "testToken",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATED,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATED,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETED,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_UPDATE_PENDING,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_CREATE_PENDING,
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_DELETE_PENDING,
				},
			},
		},
	}
	subClient, err := c.SubscribeToEvents(ctx)
	require.NoError(t, err)
	defer func() {
		err2 := subClient.CloseSend()
		require.NoError(t, err2)
	}()
	err = subClient.Send(createResourceSub)
	require.NoError(t, err)

	ev, err := subClient.Recv()
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

	powerTest := 654321
	_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": powerTest,
			}),
		},
		Force: true,
		Async: true,
	})
	require.NoError(t, err)

	_, err = c.CreateResource(ctx, &pb.CreateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesHref),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data:        test.EncodeToCbor(t, test.MakeSwitchResourceDefaultData()),
		},
		Force: true,
		Async: true,
	})
	require.NoError(t, err)

	_, err = c.DeleteResource(ctx, &pb.DeleteResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Force:      true,
		Async:      true,
	})
	require.NoError(t, err)

	pendingCommandsClient, err := c.GetPendingCommands(ctx, &pb.GetPendingCommandsRequest{
		DeviceIdFilter:         []string{deviceID},
		IncludeHiddenResources: true,
	})
	require.NoError(t, err)
	numPendingCommands := 0
	for {
		ev, err2 := pendingCommandsClient.Recv()
		if errors.Is(err2, io.EOF) {
			break
		}
		require.NoError(t, err2)
		if ev.GetResourceCreatePending() != nil {
			require.Equal(t, deviceID, ev.GetResourceCreatePending().GetResourceId().GetDeviceId())
			require.Equal(t, test.TestResourceSwitchesHref, ev.GetResourceCreatePending().GetResourceId().GetHref())
			numPendingCommands++
		}
		if ev.GetResourceUpdatePending() != nil {
			require.Equal(t, deviceID, ev.GetResourceUpdatePending().GetResourceId().GetDeviceId())
			switch ev.GetResourceUpdatePending().GetResourceId().GetHref() {
			case test.TestResourceLightInstanceHref("1"):
				numPendingCommands++
			default:
				require.FailNowf(t, "unexpected pending command", "%v", ev)
			}
		}
		if ev.GetResourceDeletePending() != nil {
			require.Equal(t, deviceID, ev.GetResourceDeletePending().GetResourceId().GetDeviceId())
			require.Equal(t, test.TestResourceLightInstanceHref("1"), ev.GetResourceDeletePending().GetResourceId().GetHref())
			numPendingCommands++
		}
	}
	require.Equal(t, 3, numPendingCommands)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, nil)
	defer shutdownDevSim()

	var lightChanged *events.ResourceChanged
	var switchChanged *events.ResourceChanged
	var lightUpdated *events.ResourceUpdated
	var switchCreated *events.ResourceCreated
	var lightDeleted *events.ResourceDeleted
	for {
		ev, err2 := subClient.Recv()
		require.NoError(t, err2)
		if ch := ev.GetResourceChanged(); ch != nil {
			if ch.GetResourceId().GetHref() == test.TestResourceLightInstanceHref("1") {
				d := test.DecodeCbor(t, ch.GetContent().GetData())
				if m, ok := d.(map[interface{}]interface{}); ok && m["power"] == uint64(powerTest) {
					lightChanged = ch
				}
			}
			if ch.GetResourceId().GetHref() == test.TestResourceSwitchesInstanceHref(switchID) {
				switchChanged = ch
			}
		}
		if updated := ev.GetResourceUpdated(); updated != nil {
			if updated.GetResourceId().GetHref() == test.TestResourceLightInstanceHref("1") {
				lightUpdated = updated
			}
		}
		if created := ev.GetResourceCreated(); created != nil {
			if created.GetResourceId().GetHref() == test.TestResourceSwitchesHref {
				switchCreated = created
			}
		}
		if deleted := ev.GetResourceDeleted(); deleted != nil {
			if deleted.GetResourceId().GetHref() == test.TestResourceLightInstanceHref("1") {
				lightDeleted = deleted
			}
		}
		if lightChanged != nil && switchChanged != nil && lightUpdated != nil && switchCreated != nil && lightDeleted != nil {
			break
		}
	}

	_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, "/not/existing/resource"),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": powerTest,
			}),
		},
		Force: true,
		Async: true,
	})
	require.NoError(t, err)

	pendingCommandsClient, err = c.GetPendingCommands(ctx, &pb.GetPendingCommandsRequest{
		DeviceIdFilter:         []string{deviceID},
		IncludeHiddenResources: true,
	})
	require.NoError(t, err)
	numPendingCommands = 0
	for {
		ev, err2 := pendingCommandsClient.Recv()
		if errors.Is(err2, io.EOF) {
			break
		}
		require.NoError(t, err2)
		if ev.GetResourceUpdatePending() != nil {
			require.Equal(t, deviceID, ev.GetResourceUpdatePending().GetResourceId().GetDeviceId())
			switch ev.GetResourceUpdatePending().GetResourceId().GetHref() {
			case "/not/existing/resource":
				numPendingCommands++
			default:
				require.FailNowf(t, "unexpected pending command", "%v", ev)
			}
		} else {
			require.FailNowf(t, "unexpected pending command", "%v", ev)
		}
	}
	require.Equal(t, 1, numPendingCommands)

	_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 0,
			}),
		},
	})
	require.NoError(t, err)
}
