package service

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/go-ocf/sdk/schema/cloud"
	"github.com/kelseyhightower/envconfig"
	"github.com/panjf2000/ants"

	cbor "github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/security/certManager"

	"github.com/go-ocf/cloud/resource-aggregate/cqrs"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventbus/nats"
	mockEventStore "github.com/go-ocf/cloud/resource-aggregate/cqrs/eventstore/test"
	mockEvents "github.com/go-ocf/cloud/resource-aggregate/cqrs/eventstore/test"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbDD "github.com/go-ocf/cloud/resource-directory/pb/device-directory"
	coap "github.com/go-ocf/go-coap"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestDeviceDirectory_GetDevices(t *testing.T) {
	type args struct {
		request pbDD.GetDevicesRequest
	}
	tests := []struct {
		name           string
		args           args
		wantResponse   map[string]*pbDD.Device
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "project_ONLINE",
			args: args{
				request: pbDD.GetDevicesRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					StatusFilter:         []pbDD.Status{pbDD.Status_ONLINE},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pbDD.Device{
				ddResource2.Resource.DeviceId: {
					Id:       ddResource2.Resource.DeviceId,
					Resource: testMakeDeviceResouceProtobuf(ddResource2.Resource.DeviceId, deviceResourceTypes),
					IsOnline: true,
				},
				ddResource4.Resource.DeviceId: {
					Id:       ddResource4.Resource.DeviceId,
					IsOnline: true,
				},
			},
		},

		{
			name: "project_OFFLINE",
			args: args{
				request: pbDD.GetDevicesRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					StatusFilter:         []pbDD.Status{pbDD.Status_OFFLINE},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pbDD.Device{
				ddResource1.Resource.DeviceId: {
					Id:       ddResource1.Resource.DeviceId,
					Resource: testMakeDeviceResouceProtobuf(ddResource1.Resource.DeviceId, deviceResourceTypes),
					IsOnline: false,
				},
			},
		},
		{
			name: "project_ONLINE_OFFLINE",
			args: args{
				request: pbDD.GetDevicesRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pbDD.Device{
				ddResource1.Resource.DeviceId: {
					Id:       ddResource1.Resource.DeviceId,
					Resource: testMakeDeviceResouceProtobuf(ddResource1.Resource.DeviceId, deviceResourceTypes),
					IsOnline: false,
				},
				ddResource2.Resource.DeviceId: {
					Id:       ddResource2.Resource.DeviceId,
					Resource: testMakeDeviceResouceProtobuf(ddResource2.Resource.DeviceId, deviceResourceTypes),
					IsOnline: true,
				},
				ddResource4.Resource.DeviceId: {
					Id:       ddResource4.Resource.DeviceId,
					IsOnline: true,
				},
			},
		},
		{
			name: "project_type_filter",
			args: args{
				request: pbDD.GetDevicesRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					TypeFilter:           []string{"customType"},
				},
			},
			wantStatusCode: codes.NotFound,
			wantErr:        true,
		},
		{
			name: "project_one_device",
			args: args{
				request: pbDD.GetDevicesRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					DeviceIdsFilter:      []string{ddResource1.Resource.DeviceId},
				},
			},
			wantStatusCode: codes.OK,
			wantResponse: map[string]*pbDD.Device{
				ddResource1.Resource.DeviceId: {
					Id:       ddResource1.Resource.DeviceId,
					Resource: testMakeDeviceResouceProtobuf(ddResource1.Resource.DeviceId, deviceResourceTypes),
					IsOnline: false,
				},
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
	resourceSubscriber, err := nats.NewSubscriber(natsCfg, pool.Submit, func(err error) { require.NoError(t, err) }, nats.WithTLS(&tlsConfig))
	require.NoError(t, err)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")
	resourceProjection, err := NewProjection(ctx, "test", testCreateResourceDeviceEventstores(), resourceSubscriber, time.Second)
	require.NoError(t, err)

	rd := NewDeviceDirectory(resourceProjection, []string{
		ddResource0.DeviceId,
		ddResource1.DeviceId,
		ddResource2.DeviceId,
		ddResource4.DeviceId,
	})

	for _, tt := range tests {
		fn := func(t *testing.T) {
			var got map[string]*pbDD.Device
			statusCode, err := rd.GetDevices(context.Background(), &tt.args.request, func(device *pbDD.Device) error {
				if got == nil {
					got = make(map[string]*pbDD.Device)
				}
				got[device.Id] = device
				return nil
			})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantStatusCode, statusCode)
			require.Equal(t, tt.wantResponse, got)
		}
		t.Run(tt.name, fn)
	}
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
		Id:            cqrs.MakeResourceId(deviceId, href),
		DeviceId:      deviceId,
		Href:          href,
		ResourceTypes: types,
	}
}

func testMakeDeviceResouceProtobuf(deviceId string, types []string) *pbDD.Resource {
	return &pbDD.Resource{
		ResourceTypes: types,
		Name:          "Name." + deviceId,
		ManufacturerName: []*pbDD.LocalizedString{
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
	}
}

var deviceResourceTypes = []string{"oic.wk.d", "x.test.d"}

func testMakeDeviceResourceContent(deviceId string) pbRA.Content {
	dr := testMakeDeviceResouceProtobuf(deviceId, deviceResourceTypes)

	d, err := cbor.Encode(dr)
	if err != nil {
		log.Fatalf("cannot decode content: %v", err)
	}

	return pbRA.Content{
		Data:        d,
		ContentType: coap.AppCBOR.String(),
	}
}

func testMakeCloudResourceContent(deviceId string, online bool) pbRA.Content {
	s := cloud.Status{
		ResourceTypes: cloud.StatusResourceTypes,
		Interfaces:    cloud.StatusInterfaces,
		Online:        online,
	}
	d, err := cbor.Encode(s)
	if err != nil {
		log.Fatalf("cannot decode content: %v", err)
	}

	return pbRA.Content{
		Data:        d,
		ContentType: coap.AppCBOR.String(),
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
		Resource: testMakeDeviceResource(deviceId, cloud.StatusHref, cloud.StatusResourceTypes),
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
	resourceEventStore.Append(ddResource0.DeviceId, ddResource0.Id, mockEvents.MakeResourcePublishedEvent(ddResource0.Resource, cqrs.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource0.DeviceId, ddResource0.Id, mockEvents.MakeResourceChangedEvent(ddResource0.Id, ddResource0.DeviceId, ddResource0.Content, cqrs.MakeEventMeta("a", 0, 1)))

	resourceEventStore.Append(ddResource1.DeviceId, ddResource1.Id, mockEvents.MakeResourcePublishedEvent(ddResource1.Resource, cqrs.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource1.DeviceId, ddResource1.Id, mockEvents.MakeResourceChangedEvent(ddResource1.Id, ddResource1.DeviceId, ddResource1.Content, cqrs.MakeEventMeta("a", 0, 1)))
	resourceEventStore.Append(ddResource1Cloud.DeviceId, ddResource1Cloud.Id, mockEvents.MakeResourcePublishedEvent(ddResource1Cloud.Resource, cqrs.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource1Cloud.DeviceId, ddResource1Cloud.Id, mockEvents.MakeResourceChangedEvent(ddResource1Cloud.Id, ddResource1Cloud.DeviceId, ddResource1Cloud.Content, cqrs.MakeEventMeta("a", 0, 1)))

	//with cloud state - online
	resourceEventStore.Append(ddResource2.DeviceId, ddResource2.Id, mockEvents.MakeResourcePublishedEvent(ddResource2.Resource, cqrs.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource2.DeviceId, ddResource2.Id, mockEvents.MakeResourceChangedEvent(ddResource2.Id, ddResource2.DeviceId, ddResource2.Content, cqrs.MakeEventMeta("a", 0, 1)))
	resourceEventStore.Append(ddResource2Cloud.DeviceId, ddResource2Cloud.Id, mockEvents.MakeResourcePublishedEvent(ddResource2Cloud.Resource, cqrs.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource2Cloud.DeviceId, ddResource2Cloud.Id, mockEvents.MakeResourceChangedEvent(ddResource2Cloud.Id, ddResource2Cloud.DeviceId, ddResource2Cloud.Content, cqrs.MakeEventMeta("a", 0, 1)))

	//without device resource
	resourceEventStore.Append(ddResource4Cloud.DeviceId, ddResource4Cloud.Id, mockEvents.MakeResourcePublishedEvent(ddResource4Cloud.Resource, cqrs.MakeEventMeta("a", 0, 0)))
	resourceEventStore.Append(ddResource4Cloud.DeviceId, ddResource4Cloud.Id, mockEvents.MakeResourceChangedEvent(ddResource4Cloud.Id, ddResource4Cloud.DeviceId, ddResource4Cloud.Content, cqrs.MakeEventMeta("a", 0, 1)))

	return resourceEventStore
}
