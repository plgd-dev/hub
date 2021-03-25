package service_test

import (
	"context"
	"log"
	"testing"
	"time"

	"google.golang.org/grpc/status"

	"github.com/kelseyhightower/envconfig"
	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/cloud/resource-directory/service"
	"github.com/plgd-dev/go-coap/v2/message"

	cbor "github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/security/certManager"

	deviceStatus "github.com/plgd-dev/cloud/coap-gateway/schema/device/status"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	mockEventStore "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	mockEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils/notification"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/stretchr/testify/assert"
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
				ddResource4.Resource.DeviceId: {
					Id:       ddResource4.Resource.DeviceId,
					IsOnline: true,
				},
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
				ddResource4.Resource.DeviceId: {
					Id:       ddResource4.Resource.DeviceId,
					IsOnline: true,
				},
			},
		},
		{
			name: "project_type_filter-not-found",
			args: args{
				request: pb.GetDevicesRequest{
					TypeFilter: []string{"notFound"},
				},
			},
			wantStatusCode: codes.NotFound,
			wantErr:        true,
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
					DeviceIdsFilter: []string{ddResource1.Resource.DeviceId},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pb.Device{
				ddResource1.Resource.DeviceId: testMakeDeviceResouceProtobuf(ddResource1.Resource.DeviceId, deviceResourceTypes, false),
			},
		},
	}
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	defer dialCertManager.Close()
	tlsConfig := dialCertManager.GetClientTLSConfig()

	pool, err := ants.NewPool(1)
	require.NoError(t, err)
	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	require.NoError(t, err)
	resourceSubscriber, err := nats.NewSubscriber(natsCfg, pool.Submit, func(err error) { require.NoError(t, err) }, nats.WithTLS(tlsConfig))
	require.NoError(t, err)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")

	subscriptions := service.NewSubscriptions()
	updateNotificationContainer := notification.NewUpdateNotificationContainer()
	retrieveNotificationContainer := notification.NewRetrieveNotificationContainer()
	deleteNotificationContainer := notification.NewDeleteNotificationContainer()

	resourceProjection, err := service.NewProjection(ctx, "test", testCreateResourceDeviceEventstores(), resourceSubscriber, service.NewResourceCtx(subscriptions, updateNotificationContainer, retrieveNotificationContainer, deleteNotificationContainer), time.Second)
	require.NoError(t, err)

	rd := service.NewDeviceDirectory(resourceProjection, []string{
		ddResource0.DeviceId,
		ddResource1.DeviceId,
		ddResource2.DeviceId,
		ddResource4.DeviceId,
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
	pbRA.Resource
	pbRA.Content
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

func testMakeDeviceResource(deviceId string, href string, types []string) pbRA.Resource {
	return pbRA.Resource{
		Id:            utils.MakeResourceId(deviceId, href),
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
		IsOnline:    isOnline,
	}
}

var deviceResourceTypes = []string{"oic.wk.d", "x.test.d"}

func testMakeDeviceResourceContent(deviceId string) pbRA.Content {
	dr := testMakeDeviceResouceProtobuf(deviceId, deviceResourceTypes, false).ToSchema()

	d, err := cbor.Encode(dr)
	if err != nil {
		log.Fatalf("cannot decode content: %v", err)
	}

	return pbRA.Content{
		Data:        d,
		ContentType: message.AppCBOR.String(),
	}
}

func testMakeCloudResourceContent(deviceId string, online bool) pbRA.Content {
	state := deviceStatus.State_Online
	if !online {
		state = deviceStatus.State_Offline
	}
	s := deviceStatus.Status{
		ResourceTypes: deviceStatus.ResourceTypes,
		Interfaces:    deviceStatus.Interfaces,
		State:         state,
	}
	d, err := cbor.Encode(s)
	if err != nil {
		log.Fatalf("cannot decode content: %v", err)
	}

	return pbRA.Content{
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

func makeTestCloudResourceContent(deviceId string, online bool) ResourceContent {
	return ResourceContent{
		Resource: testMakeDeviceResource(deviceId, deviceStatus.Href, deviceStatus.ResourceTypes),
		Content:  testMakeCloudResourceContent(deviceId, online),
	}
}

var ddResource0 = makeTestDeviceResourceContent("0")

var ddResource1 = makeTestDeviceResourceContent("1")
var ddResource1Cloud = makeTestCloudResourceContent("1", false)

var ddResource2 = makeTestDeviceResourceContent("2")
var ddResource2Cloud = makeTestCloudResourceContent("2", true)

var ddResource4 = makeTestDeviceResourceContent("4")
var ddResource4Cloud = makeTestCloudResourceContent("4", true)

func testCreateResourceDeviceEventstores() (resourceEventStore *mockEventStore.MockEventStore) {
	resourceEventStore = mockEventStore.NewMockEventStore()

	//without cloud state
	resourceEventStore.Append(ddResource0.DeviceId, ddResource0.Id, mockEvents.MakeResourcePublishedEvent(ddResource0.Resource, utils.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource0.DeviceId, ddResource0.Id, mockEvents.MakeResourceChangedEvent(ddResource0.Id, ddResource0.DeviceId, ddResource0.Content, utils.MakeEventMeta("a", 0, 1)))

	resourceEventStore.Append(ddResource1.DeviceId, ddResource1.Id, mockEvents.MakeResourcePublishedEvent(ddResource1.Resource, utils.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource1.DeviceId, ddResource1.Id, mockEvents.MakeResourceChangedEvent(ddResource1.Id, ddResource1.DeviceId, ddResource1.Content, utils.MakeEventMeta("a", 0, 1)))
	resourceEventStore.Append(ddResource1Cloud.DeviceId, ddResource1Cloud.Id, mockEvents.MakeResourcePublishedEvent(ddResource1Cloud.Resource, utils.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource1Cloud.DeviceId, ddResource1Cloud.Id, mockEvents.MakeResourceChangedEvent(ddResource1Cloud.Id, ddResource1Cloud.DeviceId, ddResource1Cloud.Content, utils.MakeEventMeta("a", 0, 1)))

	//with cloud state - online
	resourceEventStore.Append(ddResource2.DeviceId, ddResource2.Id, mockEvents.MakeResourcePublishedEvent(ddResource2.Resource, utils.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource2.DeviceId, ddResource2.Id, mockEvents.MakeResourceChangedEvent(ddResource2.Id, ddResource2.DeviceId, ddResource2.Content, utils.MakeEventMeta("a", 0, 1)))
	resourceEventStore.Append(ddResource2Cloud.DeviceId, ddResource2Cloud.Id, mockEvents.MakeResourcePublishedEvent(ddResource2Cloud.Resource, utils.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource2Cloud.DeviceId, ddResource2Cloud.Id, mockEvents.MakeResourceChangedEvent(ddResource2Cloud.Id, ddResource2Cloud.DeviceId, ddResource2Cloud.Content, utils.MakeEventMeta("a", 0, 1)))

	//without device resource
	resourceEventStore.Append(ddResource4Cloud.DeviceId, ddResource4Cloud.Id, mockEvents.MakeResourcePublishedEvent(ddResource4Cloud.Resource, utils.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource4Cloud.DeviceId, ddResource4Cloud.Id, mockEvents.MakeResourceChangedEvent(ddResource4Cloud.Id, ddResource4Cloud.DeviceId, ddResource4Cloud.Content, utils.MakeEventMeta("a", 0, 1)))

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
