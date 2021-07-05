package service_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/status"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/cloud/resource-directory/service"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/plgd-dev/go-coap/v2/message"

	cbor "github.com/plgd-dev/kit/codec/cbor"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	mockEventStore "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	mockEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils/notification"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestDeviceDirectory_GetDevices(t *testing.T) {
	type args struct {
		request pb.GetDevicesRequest
	}
	tests := []struct {
		name           string
		args           args
		wantResponse   map[string]*pb.Device
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "project_ONLINE",
			args: args{
				request: pb.GetDevicesRequest{
					StatusFilter: []pb.GetDevicesRequest_Status{pb.GetDevicesRequest_ONLINE},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pb.Device{
				ddResource2.Resource.DeviceId: testMakeDeviceResouceProtobuf(ddResource2.Resource.DeviceId, deviceResourceTypes, true),
			},
		},

		{
			name: "project_OFFLINE",
			args: args{
				request: pb.GetDevicesRequest{
					StatusFilter: []pb.GetDevicesRequest_Status{pb.GetDevicesRequest_OFFLINE},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pb.Device{
				ddResource1.Resource.DeviceId: testMakeDeviceResouceProtobuf(ddResource1.Resource.DeviceId, deviceResourceTypes, false),
			},
		},
		{
			name:           "project_ONLINE_OFFLINE",
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pb.Device{
				ddResource1.Resource.DeviceId: testMakeDeviceResouceProtobuf(ddResource1.Resource.DeviceId, deviceResourceTypes, false),
				ddResource2.Resource.DeviceId: testMakeDeviceResouceProtobuf(ddResource2.Resource.DeviceId, deviceResourceTypes, true),
			},
		},
		{
			name: "project_type_filter-not-found",
			args: args{
				request: pb.GetDevicesRequest{
					TypeFilter: []string{"notFound"},
				},
			},
			wantStatusCode: codes.OK,
		},
		{
			name: "project_type_filter",
			args: args{
				request: pb.GetDevicesRequest{
					TypeFilter: []string{"x.test.d"},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pb.Device{
				ddResource1.Resource.DeviceId: testMakeDeviceResouceProtobuf(ddResource1.Resource.DeviceId, deviceResourceTypes, false),
				ddResource2.Resource.DeviceId: testMakeDeviceResouceProtobuf(ddResource2.Resource.DeviceId, deviceResourceTypes, true),
			},
		},
		{
			name: "project_one_device",
			args: args{
				request: pb.GetDevicesRequest{
					DeviceIdFilter: []string{ddResource1.Resource.DeviceId},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pb.Device{
				ddResource1.Resource.DeviceId: testMakeDeviceResouceProtobuf(ddResource1.Resource.DeviceId, deviceResourceTypes, false),
			},
		},
	}

	logger, err := log.NewLogger(log.Config{})
	require.NoError(t, err)
	pool, err := ants.NewPool(1)
	require.NoError(t, err)
	resourceSubscriber, err := subscriber.New(config.MakeSubscriberConfig(), logger, subscriber.WithGoPool(pool.Submit))
	require.NoError(t, err)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")

	subscriptions := service.NewSubscriptions()
	updateNotificationContainer := notification.NewUpdateNotificationContainer()
	retrieveNotificationContainer := notification.NewRetrieveNotificationContainer()
	deleteNotificationContainer := notification.NewDeleteNotificationContainer()
	createNotificationContainer := notification.NewCreateNotificationContainer()

	resourceProjection, err := service.NewProjection(ctx, "test", testCreateResourceDeviceEventstores(), resourceSubscriber, service.NewEventStoreModelFactory(subscriptions, updateNotificationContainer, retrieveNotificationContainer, deleteNotificationContainer, createNotificationContainer), time.Second)
	require.NoError(t, err)

	rd := service.NewDeviceDirectory(resourceProjection, []string{
		ddResource0.Resource.GetDeviceId(),
		ddResource1.Resource.GetDeviceId(),
		ddResource2.Resource.GetDeviceId(),
		ddResource4.Resource.GetDeviceId(),
	})

	for _, tt := range tests {
		fn := func(t *testing.T) {
			var s testGrpcGateway_GetDevicesServer
			err := rd.GetDevices(&tt.args.request, &s)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantStatusCode, status.Convert(err).Code())
			require.Equal(t, tt.wantResponse, s.got)
		}
		t.Run(tt.name, fn)
	}
}

type ResourceContent struct {
	commands.Resource
	commands.Content
}

func (c ResourceContent) ToResourceIDString() string {
	return c.GetResourceID().ToString()
}

type testGeneratePublishEvent struct {
	version uint64
	ResourceContent
}

type testGenerateUnpublishEvent struct {
	version  uint64
	id       string
	deviceID string
}

type testGenerateUnknownEvent struct {
	version uint64
	id      string
}

func testMakeDeviceResource(deviceId string, href string, types []string) commands.Resource {
	return commands.Resource{
		DeviceId:      deviceId,
		Href:          href,
		ResourceTypes: types,
	}
}

func testMakeDeviceResouceProtobuf(deviceId string, types []string, isOnline bool) *pb.Device {
	return &pb.Device{
		Id:    deviceId,
		Types: types,
		Name:  "Name." + deviceId,
		ManufacturerName: []*pb.LocalizedString{
			{
				Language: "en",
				Value:    "test device resource",
			},
			{
				Language: "sk",
				Value:    "testovaci prostriedok pre zariadenie",
			},
		},
		ModelNumber: "ModelNumber." + deviceId,
		Metadata: &pb.Device_Metadata{
			Status: &commands.ConnectionStatus{
				Value: func() commands.ConnectionStatus_Status {
					if isOnline {
						return commands.ConnectionStatus_ONLINE
					}
					return commands.ConnectionStatus_OFFLINE
				}(),
			},
		},
	}
}

var deviceResourceTypes = []string{"oic.wk.d", "x.test.d"}

func testMakeDeviceResourceContent(deviceId string) commands.Content {
	dr := testMakeDeviceResouceProtobuf(deviceId, deviceResourceTypes, false).ToSchema()

	d, err := cbor.Encode(dr)
	if err != nil {
		log.Fatalf("cannot decode content: %v", err)
	}

	return commands.Content{
		Data:        d,
		ContentType: message.AppCBOR.String(),
	}
}

func makeTestDeviceResourceContent(deviceId string) ResourceContent {
	return ResourceContent{
		Resource: testMakeDeviceResource(deviceId, "/oic/d", []string{"oic.wk.d", "x.test.d"}),
		Content:  testMakeDeviceResourceContent(deviceId),
	}
}

func makeTestDeviceConnectionStatus(deviceId string, online bool) *events.DeviceMetadataUpdated {
	v := commands.ConnectionStatus_OFFLINE
	if online {
		v = commands.ConnectionStatus_ONLINE
	}
	return &events.DeviceMetadataUpdated{
		DeviceId: deviceId,
		Status: &commands.ConnectionStatus{
			Value: v,
		},
	}
}

var ddResource0 = makeTestDeviceResourceContent("0")

var ddResource1 = makeTestDeviceResourceContent("1")
var ddResource1Cloud = makeTestDeviceConnectionStatus("1", false)

var ddResource2 = makeTestDeviceResourceContent("2")
var ddResource2Cloud = makeTestDeviceConnectionStatus("2", true)

var ddResource4 = makeTestDeviceResourceContent("4")
var ddResource4Cloud = makeTestDeviceConnectionStatus("4", true)

func testCreateResourceDeviceEventstores() (resourceEventStore *mockEventStore.MockEventStore) {
	resourceEventStore = mockEventStore.NewMockEventStore()

	//without cloud state
	resourceEventStore.Append(ddResource0.DeviceId, commands.MakeLinksResourceUUID(ddResource0.DeviceId), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&ddResource0.Resource}, ddResource0.GetDeviceId(), events.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource0.DeviceId, ddResource0.Resource.ToUUID(), mockEvents.MakeResourceChangedEvent(ddResource0.Resource.GetResourceID(), &ddResource0.Content, events.MakeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))

	resourceEventStore.Append(ddResource1.DeviceId, commands.MakeLinksResourceUUID(ddResource1.DeviceId), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&ddResource1.Resource}, ddResource1.GetDeviceId(), events.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource1.DeviceId, ddResource1.Resource.ToUUID(), mockEvents.MakeResourceChangedEvent(ddResource1.Resource.GetResourceID(), &ddResource1.Content, events.MakeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	resourceEventStore.Append(ddResource1Cloud.DeviceId, ddResource1Cloud.AggregateID(), mockEvents.MakeDeviceMetadata(ddResource1Cloud.DeviceId, ddResource1Cloud, events.MakeEventMeta("a", 0, 0)))

	//with cloud state - online
	resourceEventStore.Append(ddResource2.DeviceId, commands.MakeLinksResourceUUID(ddResource2.DeviceId), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&ddResource2.Resource}, ddResource2.Resource.GetDeviceId(), events.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource2.DeviceId, ddResource2.Resource.ToUUID(), mockEvents.MakeResourceChangedEvent(ddResource2.Resource.GetResourceID(), &ddResource2.Content, events.MakeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	resourceEventStore.Append(ddResource2Cloud.DeviceId, ddResource2Cloud.AggregateID(), mockEvents.MakeDeviceMetadata(ddResource2Cloud.DeviceId, ddResource2Cloud, events.MakeEventMeta("a", 0, 0)))

	//without device resource
	resourceEventStore.Append(ddResource4Cloud.DeviceId, ddResource4Cloud.AggregateID(), mockEvents.MakeDeviceMetadata(ddResource4Cloud.DeviceId, ddResource2Cloud, events.MakeEventMeta("a", 0, 1)))

	return resourceEventStore
}

type testGrpcGateway_GetDevicesServer struct {
	got map[string]*pb.Device
	grpc.ServerStream
}

func (s *testGrpcGateway_GetDevicesServer) Context() context.Context {
	return context.Background()
}

func (s *testGrpcGateway_GetDevicesServer) Send(d *pb.Device) error {
	if s.got == nil {
		s.got = make(map[string]*pb.Device)
	}
	s.got[d.Id] = d
	return nil
}
