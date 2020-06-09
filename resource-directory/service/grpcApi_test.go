package service_test

import (
	"context"
	"crypto/tls"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/cloud/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/schema/cloud"
)

const TEST_TIMEOUT = time.Second * 20

func TestRequestHandler_UpdateResourcesValues(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req pb.UpdateResourceValuesRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.UpdateResourceValuesResponse
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				req: pb.UpdateResourceValuesRequest{
					ResourceId: &pb.ResourceId{
						DeviceId:         deviceID,
						ResourceLinkHref: "/light/1",
					},
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 1,
						}),
					},
				},
			},
			want: &pb.UpdateResourceValuesResponse{
				Content: &pb.Content{},
				Status:  pb.Status_OK,
			},
		},
		{
			name: "valid with interface",
			args: args{
				req: pb.UpdateResourceValuesRequest{
					ResourceInterface: "oic.if.baseline",
					ResourceId: &pb.ResourceId{
						DeviceId:         deviceID,
						ResourceLinkHref: "/light/1",
					},
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 2,
						}),
					},
				},
			},
			want: &pb.UpdateResourceValuesResponse{
				Content: &pb.Content{},
				Status:  pb.Status_OK,
			},
		},
		{
			name: "revert update",
			args: args{
				req: pb.UpdateResourceValuesRequest{
					ResourceInterface: "oic.if.baseline",
					ResourceId: &pb.ResourceId{
						DeviceId:         deviceID,
						ResourceLinkHref: "/light/1",
					},
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 0,
						}),
					},
				},
			},
			want: &pb.UpdateResourceValuesResponse{
				Content: &pb.Content{},
				Status:  pb.Status_OK,
			},
		},
		{
			name: "update RO-resource",
			args: args{
				req: pb.UpdateResourceValuesRequest{
					ResourceId: &pb.ResourceId{
						DeviceId:         deviceID,
						ResourceLinkHref: "/oic/d",
					},
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
			name: "invalid ResourceLinkHref",
			args: args{
				req: pb.UpdateResourceValuesRequest{
					ResourceId: &pb.ResourceId{
						DeviceId:         deviceID,
						ResourceLinkHref: "/unknown",
					},
				},
			},
			wantErr: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	log.Setup(log.Config{
		Debug: true,
	})
	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.UpdateResourcesValues(ctx, &tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, got)
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
					ResourceId: &pb.ResourceId{
						DeviceId:         deviceID,
						ResourceLinkHref: "/light/2",
					},
				},
			},
			wantContentType: "application/vnd.ocf+cbor",
			want:            map[string]interface{}{"name": "Light", "power": uint64(0), "state": false},
		},
		{
			name: "valid /oic/d",
			args: args{
				req: pb.RetrieveResourceFromDeviceRequest{
					ResourceId: &pb.ResourceId{
						DeviceId:         deviceID,
						ResourceLinkHref: "/oic/d",
					},
				},
			},
			wantContentType: "application/vnd.ocf+cbor",
			want:            map[string]interface{}{"di": deviceID, "dmv": "ocf.res.1.3.0", "icv": "ocf.2.0.5", "n": test.TestDeviceName},
		},
		{
			name: "invalid ResourceLinkHref",
			args: args{
				req: pb.RetrieveResourceFromDeviceRequest{
					ResourceId: &pb.ResourceId{
						DeviceId:         deviceID,
						ResourceLinkHref: "/unknown",
					},
				},
			},
			wantErr: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
							Token: "testToken",
							ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
								Code: pb.Event_OperationProcessed_ErrorStatus_OK,
							},
						},
					},
				},
				{
					Type: &pb.Event_SubscriptionCanceled_{
						SubscriptionCanceled: &pb.Event_SubscriptionCanceled{
							Reason: "not supported",
						},
					},
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
							Token: "testToken",
							ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
								Code: pb.Event_OperationProcessed_ErrorStatus_OK,
							},
						},
					},
				},
				{
					Type: &pb.Event_DeviceRegistered_{
						DeviceRegistered: &pb.Event_DeviceRegistered{
							DeviceId: deviceID,
						},
					},
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
							Token: "testToken",
							ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
								Code: pb.Event_OperationProcessed_ErrorStatus_OK,
							},
						},
					},
				},
				{
					Type: &pb.Event_DeviceOnline_{
						DeviceOnline: &pb.Event_DeviceOnline{
							DeviceId: deviceID,
						},
					},
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
							Token: "testToken",
							ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
								Code: pb.Event_OperationProcessed_ErrorStatus_OK,
							},
						},
					},
				},
				test.ResourceLinkToPublishEvent(deviceID, 0, test.FindResourceLink("/light/1")),
				test.ResourceLinkToPublishEvent(deviceID, 0, test.FindResourceLink("/light/2")),
				test.ResourceLinkToPublishEvent(deviceID, 0, test.FindResourceLink("/oic/p")),
				test.ResourceLinkToPublishEvent(deviceID, 0, test.FindResourceLink("/oic/d")),
				test.ResourceLinkToPublishEvent(deviceID, 0, test.FindResourceLink(cloud.StatusHref)),
				test.ResourceLinkToPublishEvent(deviceID, 0, test.FindResourceLink("/oc/con")),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
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
					link := ev.GetResourcePublished().GetLink()
					if link != nil {
						link.InstanceId = 0
					}
					require.Contains(t, tt.want, ev)
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
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)
	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())

	client, err := c.SubscribeForEvents(ctx)
	require.NoError(t, err)

	err = client.Send(&pb.SubscribeForEvents{
		Token: "testToken",
		FilterBy: &pb.SubscribeForEvents_DevicesEvent{
			DevicesEvent: &pb.SubscribeForEvents_DevicesEventFilter{
				FilterEvents: []pb.SubscribeForEvents_DevicesEventFilter_Event{
					pb.SubscribeForEvents_DevicesEventFilter_ONLINE, pb.SubscribeForEvents_DevicesEventFilter_OFFLINE,
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
				Token: "testToken",
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
	}
	require.Equal(t, expectedEvent, ev)

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type: &pb.Event_DeviceOnline_{
			DeviceOnline: &pb.Event_DeviceOnline{
				DeviceId: deviceID,
			},
		},
	}
	require.Equal(t, expectedEvent, ev)

	err = client.Send(&pb.SubscribeForEvents{
		Token: "testToken",
		FilterBy: &pb.SubscribeForEvents_ResourceEvent{
			ResourceEvent: &pb.SubscribeForEvents_ResourceEventFilter{
				ResourceId: &pb.ResourceId{
					DeviceId:         deviceID,
					ResourceLinkHref: "/light/2",
				},
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
				Token: "testToken",
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
	}
	require.Equal(t, expectedEvent, ev)
	subContentChangedID := ev.SubscriptionId

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: subContentChangedID,
		Type: &pb.Event_ResourceContentChanged{
			ResourceContentChanged: &pb.Event_ResourceChanged{
				ResourceId: &pb.ResourceId{
					DeviceId:         deviceID,
					ResourceLinkHref: "/light/2",
				},
				Content: &pb.Content{
					ContentType: message.AppOcfCbor.String(),
					Data:        []byte("\277estate\364epower\000dnameeLight\377"),
				},
			},
		},
	}
	require.Equal(t, expectedEvent, ev)

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
				Token: "updatePending + resourceUpdated",
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
	}
	require.Equal(t, expectedEvent, ev)
	subUpdatedID := ev.SubscriptionId

	_, err = c.UpdateResourcesValues(ctx, &pb.UpdateResourceValuesRequest{
		ResourceId: &pb.ResourceId{
			DeviceId:         deviceID,
			ResourceLinkHref: "/light/2",
		},
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
						ResourceId: &pb.ResourceId{
							DeviceId:         deviceID,
							ResourceLinkHref: "/light/2",
						},
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
			}
			require.Equal(t, expectedEvent, ev)
			updCorrelationID = ev.GetResourceUpdatePending().GetCorrelationId()
		case ev.GetResourceUpdated() != nil:
			expectedEvent = &pb.Event{
				SubscriptionId: subUpdatedID,
				Type: &pb.Event_ResourceUpdated_{
					ResourceUpdated: &pb.Event_ResourceUpdated{
						ResourceId: &pb.ResourceId{
							DeviceId:         deviceID,
							ResourceLinkHref: "/light/2",
						},
						Status:        pb.Status_OK,
						CorrelationId: updCorrelationID,
					},
				},
			}
			require.Equal(t, expectedEvent, ev)
		case ev.GetResourceContentChanged() != nil:
			expectedEvent = &pb.Event{
				SubscriptionId: subContentChangedID,
				Type: &pb.Event_ResourceContentChanged{
					ResourceContentChanged: &pb.Event_ResourceChanged{
						ResourceId: &pb.ResourceId{
							DeviceId:         deviceID,
							ResourceLinkHref: "/light/2",
						},
						Content: &pb.Content{
							ContentType: message.AppOcfCbor.String(),
							Data:        []byte("\277estate\364epower\030cdnameeLight\377"),
						},
					},
				},
			}
			require.Equal(t, expectedEvent, ev)
		}
	}
	shutdownDevSim()

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		Type: &pb.Event_DeviceOffline_{
			DeviceOffline: &pb.Event_DeviceOffline{
				DeviceId: deviceID,
			},
		},
	}
	require.Equal(t, expectedEvent, ev)
}
