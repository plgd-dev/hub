package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	mockEvents "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/resource-directory/service"
	"github.com/plgd-dev/hub/v2/test/config"
	cbor "github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDeviceDirectoryGetDevices(t *testing.T) {
	type args struct {
		request *pb.GetDevicesRequest
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
				request: &pb.GetDevicesRequest{
					StatusFilter: []pb.GetDevicesRequest_Status{pb.GetDevicesRequest_ONLINE},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pb.Device{
				ddResource2.Resource.GetDeviceId(): testMakeDeviceResouceProtobuf(ddResource2.Resource.GetDeviceId(), deviceResourceTypes, true),
			},
		},

		{
			name: "project_OFFLINE",
			args: args{
				request: &pb.GetDevicesRequest{
					StatusFilter: []pb.GetDevicesRequest_Status{pb.GetDevicesRequest_OFFLINE},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pb.Device{
				ddResource1.Resource.GetDeviceId(): testMakeDeviceResouceProtobuf(ddResource1.Resource.GetDeviceId(), deviceResourceTypes, false),
			},
		},
		{
			name: "project_ONLINE_OFFLINE",
			args: args{
				request: &pb.GetDevicesRequest{
					StatusFilter: []pb.GetDevicesRequest_Status{pb.GetDevicesRequest_ONLINE, pb.GetDevicesRequest_OFFLINE},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pb.Device{
				ddResource1.Resource.GetDeviceId(): testMakeDeviceResouceProtobuf(ddResource1.Resource.GetDeviceId(), deviceResourceTypes, false),
				ddResource2.Resource.GetDeviceId(): testMakeDeviceResouceProtobuf(ddResource2.Resource.GetDeviceId(), deviceResourceTypes, true),
			},
		},
		{
			name: "project_type_filter-not-found",
			args: args{
				request: &pb.GetDevicesRequest{
					TypeFilter: []string{"notFound"},
				},
			},
			wantStatusCode: codes.OK,
		},
		{
			name: "project_type_filter",
			args: args{
				request: &pb.GetDevicesRequest{
					TypeFilter: []string{"x.test.d"},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pb.Device{
				ddResource1.Resource.GetDeviceId(): testMakeDeviceResouceProtobuf(ddResource1.Resource.GetDeviceId(), deviceResourceTypes, false),
				ddResource2.Resource.GetDeviceId(): testMakeDeviceResouceProtobuf(ddResource2.Resource.GetDeviceId(), deviceResourceTypes, true),
			},
		},
		{
			name: "project_one_device",
			args: args{
				request: &pb.GetDevicesRequest{
					DeviceIdFilter: []string{ddResource1.Resource.GetDeviceId()},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pb.Device{
				ddResource1.Resource.GetDeviceId(): testMakeDeviceResouceProtobuf(ddResource1.Resource.GetDeviceId(), deviceResourceTypes, false),
			},
		},
	}

	logger := log.NewLogger(log.MakeDefaultConfig())
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	pool, err := ants.NewPool(1)
	require.NoError(t, err)
	naClient, resourceSubscriber, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), fileWatcher,
		logger, noop.NewTracerProvider(),
		subscriber.WithGoPool(pool.Submit),
		subscriber.WithUnmarshaler(utils.Unmarshal),
	)
	require.NoError(t, err)
	defer func() {
		resourceSubscriber.Close()
		naClient.Close()
	}()

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")
	mf := service.NewEventStoreModelFactory()
	resourceProjection, err := service.NewProjection(ctx, "test", testCreateResourceDeviceEventstores(), resourceSubscriber, mf, time.Second)
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
			err := rd.GetDevices(tt.args.request, &s)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantStatusCode, status.Convert(err).Code())
			for _, device := range s.got {
				require.NotEmpty(t, device.GetData().GetContent().GetData())
				device.Data = nil
			}
			require.Equal(t, tt.wantResponse, s.got)
		}
		t.Run(tt.name, fn)
	}
}

type ResourceContent struct {
	*commands.Resource
	*commands.Content
}

func (c ResourceContent) ToResourceIDString() string {
	return c.GetResourceID().ToString()
}

func testMakeDeviceResource(deviceID, href string, types []string) *commands.Resource {
	return &commands.Resource{
		DeviceId:      deviceID,
		Href:          href,
		ResourceTypes: types,
	}
}

func testMakeDeviceResouceProtobuf(deviceID string, types []string, isOnline bool) *pb.Device {
	return &pb.Device{
		Id:    deviceID,
		Types: types,
		Name:  "Name." + deviceID,
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
		ModelNumber: "ModelNumber." + deviceID,
		Metadata: &pb.Device_Metadata{
			Connection: &commands.Connection{
				Status: func() commands.Connection_Status {
					if isOnline {
						return commands.Connection_ONLINE
					}
					return commands.Connection_OFFLINE
				}(),
			},
		},
		OwnershipStatus: pb.Device_OWNED,
	}
}

var deviceResourceTypes = []string{device.ResourceType, "x.test.d"}

func testMakeDeviceResourceContent(deviceID string) *commands.Content {
	dr := testMakeDeviceResouceProtobuf(deviceID, deviceResourceTypes, false).ToSchema()

	d, err := cbor.Encode(dr)
	if err != nil {
		log.Fatalf("cannot decode content: %v", err)
	}

	return &commands.Content{
		Data:        d,
		ContentType: message.AppCBOR.String(),
	}
}

func makeTestDeviceResourceContent(deviceID string) ResourceContent {
	return ResourceContent{
		Resource: testMakeDeviceResource(deviceID, device.ResourceURI, []string{device.ResourceType, "x.test.d"}),
		Content:  testMakeDeviceResourceContent(deviceID),
	}
}

func makeTestDeviceConnectionStatus(deviceID string, online bool) *events.DeviceMetadataUpdated {
	v := commands.Connection_OFFLINE
	if online {
		v = commands.Connection_ONLINE
	}
	return &events.DeviceMetadataUpdated{
		DeviceId: deviceID,
		Connection: &commands.Connection{
			Status: v,
		},
	}
}

var ddResource0 = makeTestDeviceResourceContent("0")

var (
	ddResource1      = makeTestDeviceResourceContent("1")
	ddResource1Cloud = makeTestDeviceConnectionStatus("1", false)
)

var (
	ddResource2      = makeTestDeviceResourceContent("2")
	ddResource2Cloud = makeTestDeviceConnectionStatus("2", true)
)

var (
	ddResource4      = makeTestDeviceResourceContent("4")
	ddResource4Cloud = makeTestDeviceConnectionStatus("4", true)
)

func testCreateResourceDeviceEventstores() (resourceEventStore *mockEvents.MockEventStore) {
	resourceEventStore = mockEvents.NewMockEventStore()

	// without cloud state
	resourceEventStore.Append(ddResource0.GetDeviceId(), commands.MakeLinksResourceUUID(ddResource0.GetDeviceId()).String(), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{ddResource0.Resource}, ddResource0.GetDeviceId(), events.MakeEventMeta("a", 0, 0, "hubID")))
	resourceEventStore.Append(ddResource0.GetDeviceId(), ddResource0.Resource.ToUUID().String(), mockEvents.MakeResourceChangedEvent(ddResource0.Resource.GetResourceID(), ddResource0.Content, events.MakeEventMeta("a", 0, 0, "hubID"), mockEvents.MakeAuditContext("userId", "0"), []string{"type0"}))

	resourceEventStore.Append(ddResource1.GetDeviceId(), commands.MakeLinksResourceUUID(ddResource1.GetDeviceId()).String(), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{ddResource1.Resource}, ddResource1.GetDeviceId(), events.MakeEventMeta("a", 0, 0, "hubID")))
	resourceEventStore.Append(ddResource1.GetDeviceId(), ddResource1.Resource.ToUUID().String(), mockEvents.MakeResourceChangedEvent(ddResource1.Resource.GetResourceID(), ddResource1.Content, events.MakeEventMeta("a", 0, 0, "hubID"), mockEvents.MakeAuditContext("userId", "0"), []string{"type1"}))
	resourceEventStore.Append(ddResource1Cloud.GetDeviceId(), ddResource1Cloud.AggregateID(), mockEvents.MakeDeviceMetadata(ddResource1Cloud.GetDeviceId(), ddResource1Cloud, events.MakeEventMeta("a", 0, 0, "hubID")))

	// with cloud state - online
	resourceEventStore.Append(ddResource2.GetDeviceId(), commands.MakeLinksResourceUUID(ddResource2.GetDeviceId()).String(), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{ddResource2.Resource}, ddResource2.Resource.GetDeviceId(), events.MakeEventMeta("a", 0, 0, "hubID")))
	resourceEventStore.Append(ddResource2.GetDeviceId(), ddResource2.Resource.ToUUID().String(), mockEvents.MakeResourceChangedEvent(ddResource2.Resource.GetResourceID(), ddResource2.Content, events.MakeEventMeta("a", 0, 0, "hubID"), mockEvents.MakeAuditContext("userId", "0"), []string{"type2"}))
	resourceEventStore.Append(ddResource2Cloud.GetDeviceId(), ddResource2Cloud.AggregateID(), mockEvents.MakeDeviceMetadata(ddResource2Cloud.GetDeviceId(), ddResource2Cloud, events.MakeEventMeta("a", 0, 0, "hubID")))

	// without device resource
	resourceEventStore.Append(ddResource4Cloud.GetDeviceId(), ddResource4Cloud.AggregateID(), mockEvents.MakeDeviceMetadata(ddResource4Cloud.GetDeviceId(), ddResource2Cloud, events.MakeEventMeta("a", 0, 1, "hubID")))

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
	s.got[d.GetId()] = d
	return nil
}
