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
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
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
			want: pbTest.MakeResourceUpdated(t, deviceID, test.TestResourceLightInstanceHref("1"), "", nil),
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
			want: pbTest.MakeResourceUpdated(t, deviceID, test.TestResourceLightInstanceHref("1"), "", nil),
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
			want: pbTest.MakeResourceUpdated(t, deviceID, test.TestResourceLightInstanceHref("1"), "", nil),
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
				Status:       commands.Status_OK,
				AuditContext: commands.NewAuditContext(oauthService.DeviceUserID, ""),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.COAPS_TCP_SCHEME+testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)
	time.Sleep(200 * time.Millisecond)

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

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.COAPS_TCP_SCHEME+testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	getData := func(devID string) map[string]interface{} {
		resources := make(map[string]interface{})
		client, err := c.GetResources(ctx, &pb.GetResourcesRequest{
			DeviceIdFilter: []string{devID},
		})
		assert.NoError(t, err)
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
		time.Sleep(time.Millisecond * 200)
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
	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT+(time.Second*3*time.Duration(numIteration)))
	defer cancel()

	raCfg := raTest.MakeConfig(t)
	raCfg.Clients.Eventstore.SnapshotThreshold = 5
	tearDown := service.SetUp(ctx, t, service.WithRAConfig(raCfg))
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resources := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.COAPS_TCP_SCHEME+testCfg.GW_HOST, resources)
	defer shutdownDevSim()

	for i := 0; i < numIteration; i++ {
		func() {
			t.Logf("TestRequestHandlerMultipleUpdateResource:run %v\n", i)
			lightHref := test.TestResourceLightInstanceHref("1")
			ctx, cancel := context.WithTimeout(ctx, time.Second*3)
			defer cancel()
			subClient, err := c.SubscribeToEvents(ctx)
			require.NoError(t, err)
			defer func() {
				err = subClient.CloseSend()
				require.NoError(t, err)
			}()

			err = subClient.Send(&pb.SubscribeToEvents{Action: &pb.SubscribeToEvents_CreateSubscription_{
				CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
					EventFilter:      []pb.SubscribeToEvents_CreateSubscription_Event{pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED},
					ResourceIdFilter: []string{commands.NewResourceID(deviceID, lightHref).ToString()},
				},
			}})
			require.NoError(t, err)
			ev, err := subClient.Recv()
			require.NoError(t, err)
			require.Equal(t, ev.GetOperationProcessed().GetErrorStatus().GetCode(), pb.Event_OperationProcessed_ErrorStatus_OK)
			for j := 1; j >= 0; j-- {
				_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
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
				pbTest.CmpResourceChanged(t, pbTest.MakeResourceChanged(t, deviceID, lightHref, "", makeLightData(j)), ev.GetResourceChanged(), "")
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
	raCfg.Clients.Eventstore.SnapshotThreshold = 5
	raCfg.Clients.Eventstore.ConcurrencyExceptionMaxRetry = 40

	coapCfg := coapTest.MakeConfig(t)
	tearDown := service.SetUp(ctx, t, service.WithRAConfig(raCfg), service.WithCOAPGWConfig(coapCfg))
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resources := test.GetAllBackendResourceLinks()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.COAPS_TCP_SCHEME+testCfg.GW_HOST, resources)
	defer shutdownDevSim()

	var wg sync.WaitGroup
	wg.Add(numIteration)
	for i := 0; i < numIteration; i++ {
		t.Logf("TestRequestHandlerRunMultipleParallelUpdateResource:run %v\n", i)
		go func() {
			defer wg.Done()
			lightHref := test.TestResourceLightInstanceHref("1")
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			for j := 1; j >= 0; j-- {
				_, err := c.UpdateResource(ctx, &pb.UpdateResourceRequest{
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
		}()
	}
	wg.Wait()
}
