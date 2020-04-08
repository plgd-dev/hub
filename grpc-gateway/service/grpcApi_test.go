package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	_ "github.com/mattn/go-sqlite3" // sql driver
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	grpcTest "github.com/go-ocf/cloud/grpc-gateway/test"
	coap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/cbor"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/schema/cloud"
)

const TEST_TIMEOUT = time.Second * 20

func TestRequestHandler_GetDevices(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
	type args struct {
		req *pb.GetDevicesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.Device
	}{
		{
			name: "valid",
			args: args{
				req: &pb.GetDevicesRequest{},
			},
			want: []*pb.Device{
				{
					Id:       deviceID,
					Name:     grpcTest.TestDeviceName,
					IsOnline: true,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(grpcTest.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: grpcTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c, deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetDevices(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				devices := make([]*pb.Device, 0, 1)
				for {
					dev, err := client.Recv()
					if err == io.EOF {
						break
					}
					require.NoError(t, err)
					devices = append(devices, dev)
				}
				require.Equal(t, tt.want, devices)
			}
		})
	}
}

func TestRequestHandler_GetResourceLinks(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
	type args struct {
		req *pb.GetResourceLinksRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []pb.ResourceLink
	}{
		{
			name: "valid",
			args: args{
				req: &pb.GetResourceLinksRequest{},
			},
			wantErr: false,
			want:    grpcTest.SortResources(grpcTest.ConvertSchemaToPb(deviceID, grpcTest.GetAllBackendResourceLinks())),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(grpcTest.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: grpcTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c, deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetResourceLinks(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				links := make([]pb.ResourceLink, 0, 1)
				for {
					link, err := client.Recv()
					if err == io.EOF {
						break
					}
					require.NoError(t, err)
					links = append(links, *link)
				}
				require.Equal(t, tt.want, grpcTest.SortResources(links))
			}
		})
	}
}

func cmpResourceValues(t *testing.T, want []*pb.ResourceValue, got []*pb.ResourceValue) {
	require.Len(t, got, len(want))
	for idx := range want {
		dataWant := want[idx].GetContent().GetData()
		datagot := got[idx].GetContent().GetData()
		want[idx].Content.Data = nil
		got[idx].Content.Data = nil
		require.Equal(t, want[idx], got[idx])
		w := grpcTest.DecodeCbor(t, dataWant)
		g := grpcTest.DecodeCbor(t, datagot)
		require.Equal(t, w, g)
	}
}

func TestRequestHandler_RetrieveResourcesValues(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
	type args struct {
		req *pb.RetrieveResourcesValuesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.ResourceValue
	}{
		{
			name: "valid",
			args: args{
				req: &pb.RetrieveResourcesValuesRequest{
					ResourceIdsFilter: []*pb.ResourceId{
						{
							DeviceId:         deviceID,
							ResourceLinkHref: cloud.StatusHref,
						},
					},
				},
			},
			want: []*pb.ResourceValue{
				{
					ResourceId: &pb.ResourceId{
						DeviceId:         deviceID,
						ResourceLinkHref: cloud.StatusHref,
					},
					Types: cloud.StatusResourceTypes,
					Content: &pb.Content{
						ContentType: coap.AppOcfCbor.String(),
						Data: grpcTest.EncodeToCbor(t, map[string]interface{}{
							"if":     cloud.StatusInterfaces,
							"rt":     cloud.StatusResourceTypes,
							"online": true,
						}),
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(grpcTest.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: grpcTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c, deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.RetrieveResourcesValues(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				values := make([]*pb.ResourceValue, 0, 1)
				for {
					value, err := client.Recv()
					if err == io.EOF {
						break
					}
					require.NoError(t, err)
					values = append(values, value)
				}
				cmpResourceValues(t, tt.want, values)
			}
		})
	}
}

func TestRequestHandler_UpdateResourcesValues(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
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
						ContentType: coap.AppOcfCbor.String(),
						Data: grpcTest.EncodeToCbor(t, map[string]interface{}{
							"power": 1,
						}),
					},
				},
			},
			want: &pb.UpdateResourceValuesResponse{
				Content: &pb.Content{},
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
						ContentType: coap.AppOcfCbor.String(),
						Data: grpcTest.EncodeToCbor(t, map[string]interface{}{
							"power": 2,
						}),
					},
				},
			},
			want: &pb.UpdateResourceValuesResponse{
				Content: &pb.Content{},
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
						ContentType: coap.AppOcfCbor.String(),
						Data: grpcTest.EncodeToCbor(t, map[string]interface{}{
							"power": 0,
						}),
					},
				},
			},
			want: &pb.UpdateResourceValuesResponse{
				Content: &pb.Content{},
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
						ContentType: coap.AppOcfCbor.String(),
						Data: grpcTest.EncodeToCbor(t, map[string]interface{}{
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

	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(grpcTest.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: grpcTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c, deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())
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
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
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
			want:            map[string]interface{}{"di": deviceID, "dmv": "ocf.res.1.0.0", "icv": "ocf.1.0.0", "n": grpcTest.TestDeviceName},
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

	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(grpcTest.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: grpcTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c, deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())
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
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
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
				{
					Type: &pb.Event_ResourcePublished_{
						ResourcePublished: &pb.Event_ResourcePublished{
							Link: &pb.ResourceLink{
								Href:       "/light/1",
								Types:      []string{"core.light"},
								Interfaces: []string{"oic.if.rw", "oic.if.baseline"},
								DeviceId:   deviceID,
							},
						},
					},
				},
				{
					Type: &pb.Event_ResourcePublished_{
						ResourcePublished: &pb.Event_ResourcePublished{
							Link: &pb.ResourceLink{
								Href:       "/light/2",
								Types:      []string{"core.light"},
								Interfaces: []string{"oic.if.rw", "oic.if.baseline"},
								DeviceId:   deviceID,
							},
						},
					},
				},
				{
					Type: &pb.Event_ResourcePublished_{
						ResourcePublished: &pb.Event_ResourcePublished{
							Link: &pb.ResourceLink{
								Href:       "/oic/p",
								Types:      []string{"oic.wk.p"},
								Interfaces: []string{"oic.if.r", "oic.if.baseline"},
								DeviceId:   deviceID,
							},
						},
					},
				},
				{
					Type: &pb.Event_ResourcePublished_{
						ResourcePublished: &pb.Event_ResourcePublished{
							Link: &pb.ResourceLink{
								Href:       "/oic/d",
								Types:      []string{"oic.d.cloudDevice", "oic.wk.d"},
								Interfaces: []string{"oic.if.r", "oic.if.baseline"},
								DeviceId:   deviceID,
							},
						},
					},
				},
				{
					Type: &pb.Event_ResourcePublished_{
						ResourcePublished: &pb.Event_ResourcePublished{
							Link: &pb.ResourceLink{
								Href:       cloud.StatusHref,
								Types:      cloud.StatusResourceTypes,
								Interfaces: cloud.StatusInterfaces,
								DeviceId:   deviceID,
							},
						},
					},
				},
				{
					Type: &pb.Event_ResourcePublished_{
						ResourcePublished: &pb.Event_ResourcePublished{
							Link: &pb.ResourceLink{
								Href:       "/oc/con",
								Types:      []string{"oic.wk.con"},
								Interfaces: []string{"oic.if.rw", "oic.if.baseline"},
								DeviceId:   deviceID,
							},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(grpcTest.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: grpcTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c, deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())
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
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)
	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(grpcTest.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: grpcTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c, deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())

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
					ContentType: coap.AppOcfCbor.String(),
					Data:        []byte("\277estate\364epower\000dnameeLight\377"),
				},
			},
		},
	}
	require.Equal(t, expectedEvent, ev)

	_, err = c.UpdateResourcesValues(ctx, &pb.UpdateResourceValuesRequest{
		ResourceId: &pb.ResourceId{
			DeviceId:         deviceID,
			ResourceLinkHref: "/light/2",
		},
		Content: &pb.Content{
			ContentType: coap.AppOcfCbor.String(),
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
					ContentType: coap.AppOcfCbor.String(),
					Data:        []byte("\277estate\364epower\030cdnameeLight\377"),
				},
			},
		},
	}
	require.Equal(t, expectedEvent, ev)

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
