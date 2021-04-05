package service_test

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/log"
)

const TEST_TIMEOUT = time.Second * 30

func TestRequestHandler_UpdateResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req pb.UpdateResourceRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.UpdateResourceResponse
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				req: pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/light/1"),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 1,
						}),
					},
				},
			},
			want: &pb.UpdateResourceResponse{
				Content: &pb.Content{},
				Status:  pb.Status_OK,
			},
		},
		{
			name: "valid with interface",
			args: args{
				req: pb.UpdateResourceRequest{
					ResourceInterface: "oic.if.baseline",
					ResourceId:        commands.NewResourceID(deviceID, "/light/1"),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 2,
						}),
					},
				},
			},
			want: &pb.UpdateResourceResponse{
				Content: &pb.Content{},
				Status:  pb.Status_OK,
			},
		},
		{
			name: "revert update",
			args: args{
				req: pb.UpdateResourceRequest{
					ResourceInterface: "oic.if.baseline",
					ResourceId:        commands.NewResourceID(deviceID, "/light/1"),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 0,
						}),
					},
				},
			},
			want: &pb.UpdateResourceResponse{
				Content: &pb.Content{},
				Status:  pb.Status_OK,
			},
		},
		{
			name: "update RO-resource",
			args: args{
				req: pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
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
			name: "invalid Href",
			args: args{
				req: pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/unknown"),
				},
			},
			wantErr: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	log.Setup(log.Config{
		Debug: true,
	})
	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.UpdateResource(ctx, &tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			test.CheckProtobufs(t, tt.want, got, test.RequireToCheckFunc(require.Equal))
		})
	}
}

func TestRequestHandler_RetrieveResourceFromDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req pb.RetrieveResourceFromDeviceRequest
	}
	tests := []struct {
		name            string
		args            args
		want            map[string]interface{}
		wantContentType string
		wantErr         bool
	}{
		{
			name: "valid /light/2",
			args: args{
				req: pb.RetrieveResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/light/2"),
				},
			},
			wantContentType: "application/vnd.ocf+cbor",
			want:            map[string]interface{}{"name": "Light", "power": uint64(0), "state": false},
		},
		{
			name: "valid /oic/d",
			args: args{
				req: pb.RetrieveResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
				},
			},
			wantContentType: "application/vnd.ocf+cbor",
			want:            map[string]interface{}{"di": deviceID, "dmv": "ocf.res.1.3.0", "icv": "ocf.2.0.5", "n": test.TestDeviceName},
		},
		{
			name: "invalid Href",
			args: args{
				req: pb.RetrieveResourceFromDeviceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/unknown"),
				},
			},
			wantErr: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 17; i++ {
				got, err := c.RetrieveResourceFromDevice(ctx, &tt.args.req)
				if tt.wantErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.wantContentType, got.GetContent().GetContentType())
					var d map[string]interface{}
					err := cbor.Decode(got.GetContent().GetData(), &d)
					require.NoError(t, err)
					delete(d, "piid")
					assert.Equal(t, tt.want, d)
				}
			}
		})
	}
}

func TestRequestHandler_SubscribeForEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		sub pb.SubscribeForEvents
	}
	tests := []struct {
		name string
		args args
		want []*pb.Event
	}{
		{
			name: "invalid - invalid type subscription",
			args: args{
				sub: pb.SubscribeForEvents{
					Token: "testToken",
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
					Token: "testToken",
				},
				{
					Type: &pb.Event_SubscriptionCanceled_{
						SubscriptionCanceled: &pb.Event_SubscriptionCanceled{
							Reason: "not supported",
						},
					},
					Token: "testToken",
				},
			},
		},
		{
			name: "devices subscription - registered",
			args: args{
				sub: pb.SubscribeForEvents{
					Token: "testToken",
					FilterBy: &pb.SubscribeForEvents_DevicesEvent{
						DevicesEvent: &pb.SubscribeForEvents_DevicesEventFilter{
							FilterEvents: []pb.SubscribeForEvents_DevicesEventFilter_Event{
								pb.SubscribeForEvents_DevicesEventFilter_REGISTERED, pb.SubscribeForEvents_DevicesEventFilter_UNREGISTERED,
							},
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
					Token: "testToken",
				},
				{
					Type: &pb.Event_DeviceRegistered_{
						DeviceRegistered: &pb.Event_DeviceRegistered{
							DeviceIds: []string{deviceID},
						},
					},
					Token: "testToken",
				},
			},
		},
		{
			name: "devices subscription - online",
			args: args{
				sub: pb.SubscribeForEvents{
					Token: "testToken",
					FilterBy: &pb.SubscribeForEvents_DevicesEvent{
						DevicesEvent: &pb.SubscribeForEvents_DevicesEventFilter{
							FilterEvents: []pb.SubscribeForEvents_DevicesEventFilter_Event{
								pb.SubscribeForEvents_DevicesEventFilter_ONLINE, pb.SubscribeForEvents_DevicesEventFilter_OFFLINE,
							},
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
					Token: "testToken",
				},
				{
					Type: &pb.Event_DeviceOnline_{
						DeviceOnline: &pb.Event_DeviceOnline{
							DeviceIds: []string{deviceID},
						},
					},
					Token: "testToken",
				},
			},
		},
		{
			name: "device subscription - published",
			args: args{
				sub: pb.SubscribeForEvents{
					Token: "testToken",
					FilterBy: &pb.SubscribeForEvents_DeviceEvent{
						DeviceEvent: &pb.SubscribeForEvents_DeviceEventFilter{
							DeviceId: deviceID,
							FilterEvents: []pb.SubscribeForEvents_DeviceEventFilter_Event{
								pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_PUBLISHED, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UNPUBLISHED,
							},
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
					Token: "testToken",
				},
				test.ResourceLinkToPublishEvent(deviceID, "testToken", test.GetAllBackendResourceLinks()),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.SubscribeForEvents(ctx)
			require.NoError(t, err)
			defer client.CloseSend()
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				for _, w := range tt.want {
					ev, err := client.Recv()
					require.NoError(t, err)
					ev.SubscriptionId = w.SubscriptionId
					if ev.GetResourcePublished() != nil {
						ev.GetResourcePublished().Links = test.SortResources(ev.GetResourcePublished().GetLinks())
					}
					if w.GetResourcePublished() != nil {
						w.GetResourcePublished().Links = test.SortResources(w.GetResourcePublished().GetLinks())
					}
					test.CheckProtobufs(t, tt.want, ev, test.RequireToCheckFunc(require.Contains))
				}
			}()
			err = client.Send(&tt.args.sub)
			require.NoError(t, err)
			wg.Wait()
		})
	}
}

func TestRequestHandler_ValidateEventsFlow(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())

	client, err := c.SubscribeForEvents(ctx)
	require.NoError(t, err)

	err = client.Send(&pb.SubscribeForEvents{
		Token: "testToken",
		FilterBy: &pb.SubscribeForEvents_DevicesEvent{
			DevicesEvent: &pb.SubscribeForEvents_DevicesEventFilter{
				FilterEvents: []pb.SubscribeForEvents_DevicesEventFilter_Event{
					pb.SubscribeForEvents_DevicesEventFilter_ONLINE, pb.SubscribeForEvents_DevicesEventFilter_OFFLINE, pb.SubscribeForEvents_DevicesEventFilter_REGISTERED, pb.SubscribeForEvents_DevicesEventFilter_UNREGISTERED,
				},
			},
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
		Token: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type: &pb.Event_DeviceRegistered_{
			DeviceRegistered: &pb.Event_DeviceRegistered{
				DeviceIds: []string{deviceID},
			},
		},
		Token: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type: &pb.Event_DeviceUnregistered_{
			DeviceUnregistered: &pb.Event_DeviceUnregistered{},
		},
		Token: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type: &pb.Event_DeviceOnline_{
			DeviceOnline: &pb.Event_DeviceOnline{
				DeviceIds: []string{deviceID},
			},
		},
		Token: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type: &pb.Event_DeviceOffline_{
			DeviceOffline: &pb.Event_DeviceOffline{},
		},
		Token: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	err = client.Send(&pb.SubscribeForEvents{
		Token: "testToken",
		FilterBy: &pb.SubscribeForEvents_ResourceEvent{
			ResourceEvent: &pb.SubscribeForEvents_ResourceEventFilter{
				ResourceId: commands.NewResourceID(deviceID, "/light/2"),
				FilterEvents: []pb.SubscribeForEvents_ResourceEventFilter_Event{
					pb.SubscribeForEvents_ResourceEventFilter_CONTENT_CHANGED,
				},
			},
		},
	})
	require.NoError(t, err)

	ev, err = client.Recv()
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
		Token: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
	subContentChangedID := ev.SubscriptionId

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: subContentChangedID,
		Type: &pb.Event_ResourceChanged_{
			ResourceChanged: &pb.Event_ResourceChanged{
				ResourceId: commands.NewResourceID(deviceID, "/light/2"),
				Content: &pb.Content{
					ContentType: message.AppOcfCbor.String(),
					Data: func() []byte {
						ret, err := base64.StdEncoding.DecodeString("v2JydJ9qY29yZS5saWdodP9iaWafaW9pYy5pZi5yd29vaWMuaWYuYmFzZWxpbmX/ZXN0YXRl9GVwb3dlcgBkbmFtZWVMaWdodP8=")
						require.NoError(t, err)
						return ret
					}(),
				},
				Status: pb.Status_OK,
			},
		},
		Token: "testToken",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	err = client.Send(&pb.SubscribeForEvents{
		Token: "updatePending + resourceUpdated",
		FilterBy: &pb.SubscribeForEvents_DeviceEvent{
			DeviceEvent: &pb.SubscribeForEvents_DeviceEventFilter{
				DeviceId: deviceID,
				FilterEvents: []pb.SubscribeForEvents_DeviceEventFilter_Event{
					pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATE_PENDING, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATED,
				},
			},
		},
	})
	require.NoError(t, err)

	ev, err = client.Recv()
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
		Token: "updatePending + resourceUpdated",
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
		ev, err = client.Recv()
		require.NoError(t, err)
		switch {
		case ev.GetResourceUpdatePending() != nil:
			expectedEvent = &pb.Event{
				SubscriptionId: subUpdatedID,
				Type: &pb.Event_ResourceUpdatePending_{
					ResourceUpdatePending: &pb.Event_ResourceUpdatePending{
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
						CorrelationId: ev.GetResourceUpdatePending().GetCorrelationId(),
					},
				},
				Token: "updatePending + resourceUpdated",
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
			updCorrelationID = ev.GetResourceUpdatePending().GetCorrelationId()
		case ev.GetResourceUpdated() != nil:
			expectedEvent = &pb.Event{
				SubscriptionId: subUpdatedID,
				Type: &pb.Event_ResourceUpdated_{
					ResourceUpdated: &pb.Event_ResourceUpdated{
						ResourceId:    commands.NewResourceID(deviceID, "/light/2"),
						Status:        pb.Status_OK,
						CorrelationId: updCorrelationID,
					},
				},
				Token: "updatePending + resourceUpdated",
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
		case ev.GetResourceChanged() != nil:
			expectedEvent = &pb.Event{
				SubscriptionId: subContentChangedID,
				Type: &pb.Event_ResourceChanged_{
					ResourceChanged: &pb.Event_ResourceChanged{
						ResourceId: commands.NewResourceID(deviceID, "/light/2"),
						Content: &pb.Content{
							ContentType: message.AppOcfCbor.String(),
							Data:        []byte("\277estate\364epower\030cdnameeLight\377"),
						},
						Status: pb.Status_OK,
					},
				},
				Token: "testToken",
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
	for i := 0; i < 3; i++ {
		ev, err = client.Recv()
		require.NoError(t, err)
		switch {
		case ev.GetResourceUpdatePending() != nil:
			expectedEvent = &pb.Event{
				SubscriptionId: subUpdatedID,
				Type: &pb.Event_ResourceUpdatePending_{
					ResourceUpdatePending: &pb.Event_ResourceUpdatePending{
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
						CorrelationId: ev.GetResourceUpdatePending().GetCorrelationId(),
					},
				},
				Token: "updatePending + resourceUpdated",
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
			updCorrelationID = ev.GetResourceUpdatePending().GetCorrelationId()
		case ev.GetResourceUpdated() != nil:
			expectedEvent = &pb.Event{
				SubscriptionId: subUpdatedID,
				Type: &pb.Event_ResourceUpdated_{
					ResourceUpdated: &pb.Event_ResourceUpdated{
						ResourceId:    commands.NewResourceID(deviceID, "/light/2"),
						Status:        pb.Status_OK,
						CorrelationId: updCorrelationID,
					},
				},
				Token: "updatePending + resourceUpdated",
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
		case ev.GetResourceChanged() != nil:
			expectedEvent = &pb.Event{
				SubscriptionId: subContentChangedID,
				Type: &pb.Event_ResourceChanged_{
					ResourceChanged: &pb.Event_ResourceChanged{
						ResourceId: commands.NewResourceID(deviceID, "/light/2"),
						Content: &pb.Content{
							ContentType: message.AppOcfCbor.String(),
							Data:        []byte("\277estate\364epower\000dnameeLight\377"),
						},
						Status: pb.Status_OK,
					},
				},
				Token: "testToken",
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
		}
	}

	err = client.Send(&pb.SubscribeForEvents{
		Token: "receivePending + resourceReceived",
		FilterBy: &pb.SubscribeForEvents_DeviceEvent{
			DeviceEvent: &pb.SubscribeForEvents_DeviceEventFilter{
				DeviceId: deviceID,
				FilterEvents: []pb.SubscribeForEvents_DeviceEventFilter_Event{
					pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVE_PENDING, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVED,
				},
			},
		},
	})
	require.NoError(t, err)

	ev, err = client.Recv()
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
		Token: "receivePending + resourceReceived",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
	subReceivedID := ev.SubscriptionId

	_, err = c.RetrieveResourceFromDevice(ctx, &pb.RetrieveResourceFromDeviceRequest{
		ResourceId: commands.NewResourceID(deviceID, "/light/2"),
	})
	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: subReceivedID,
		Type: &pb.Event_ResourceRetrievePending_{
			ResourceRetrievePending: &pb.Event_ResourceRetrievePending{
				ResourceId:    commands.NewResourceID(deviceID, "/light/2"),
				CorrelationId: ev.GetResourceRetrievePending().GetCorrelationId(),
			},
		},
		Token: "receivePending + resourceReceived",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
	recvCorrelationID := ev.GetResourceRetrievePending().GetCorrelationId()

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: subReceivedID,
		Type: &pb.Event_ResourceRetrieved_{
			ResourceRetrieved: &pb.Event_ResourceRetrieved{
				ResourceId: commands.NewResourceID(deviceID, "/light/2"),
				Content: &pb.Content{
					ContentType: message.AppOcfCbor.String(),
					Data:        []byte("\277estate\364epower\000dnameeLight\377"),
				},
				Status:        pb.Status_OK,
				CorrelationId: recvCorrelationID,
			},
		},
		Token: "receivePending + resourceReceived",
	}
	test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))

	shutdownDevSim()

	run := true
	for run {
		ev, err = client.Recv()
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
				Token: "testToken",
			}
			test.CheckProtobufs(t, expectedEvent, ev, test.RequireToCheckFunc(require.Equal))
			run = false
		}
	}
}
