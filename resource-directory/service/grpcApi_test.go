package service

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventbus/nats"
	mockEventStore "github.com/go-ocf/cloud/resource-aggregate/cqrs/eventstore/test"
	mockEvents "github.com/go-ocf/cloud/resource-aggregate/cqrs/eventstore/test"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbDD "github.com/go-ocf/cloud/resource-directory/pb/device-directory"
	pbRD "github.com/go-ocf/cloud/resource-directory/pb/resource-directory"
	pbRS "github.com/go-ocf/cloud/resource-directory/pb/resource-shadow"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/security/certManager"
	"github.com/panjf2000/ants"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestRequestHandler_RetrieveResourcesValues(t *testing.T) {
	type args struct {
		req *pbRS.RetrieveResourcesValuesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    map[string]*pbRS.ResourceValue
	}{
		{
			name: "list unauthorized device",
			args: args{
				req: &pbRS.RetrieveResourcesValuesRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					DeviceIdsFilter:      []string{Resource0.DeviceId},
				},
			},
			wantErr: true,
		},
		{
			name: "filter by resource Id",
			args: args{
				req: &pbRS.RetrieveResourcesValuesRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					ResourceIdsFilter:    []string{Resource1.Id, Resource2.Id},
				},
			},
			want: map[string]*pbRS.ResourceValue{
				Resource1.Id: {
					ResourceId: Resource1.Id,
					DeviceId:   Resource1.DeviceId,
					Href:       Resource1.Href,
					Content:    &Resource1.Content,
					Types:      Resource1.ResourceTypes,
				},
				Resource2.Id: {
					ResourceId: Resource2.Id,
					DeviceId:   Resource2.DeviceId,
					Href:       Resource2.Href,
					Content:    &Resource2.Content,
					Types:      Resource2.ResourceTypes,
				},
			},
		},
		{
			name: "filter by device Id",
			args: args{
				req: &pbRS.RetrieveResourcesValuesRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					DeviceIdsFilter:      []string{Resource1.DeviceId},
				},
			},
			want: map[string]*pbRS.ResourceValue{
				Resource1.Id: {
					ResourceId: Resource1.Id,
					DeviceId:   Resource1.DeviceId,
					Href:       Resource1.Href,
					Content:    &Resource1.Content,
					Types:      Resource1.ResourceTypes,
				},
				Resource3.Id: {
					ResourceId: Resource3.Id,
					DeviceId:   Resource3.DeviceId,
					Href:       Resource3.Href,
					Content:    &Resource3.Content,
					Types:      Resource3.ResourceTypes,
				},
			},
		},
		{
			name: "filter by type",
			args: args{
				req: &pbRS.RetrieveResourcesValuesRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					TypeFilter:           []string{Resource2.ResourceTypes[0]},
				},
			},
			want: map[string]*pbRS.ResourceValue{
				Resource1.Id: {
					ResourceId: Resource1.Id,
					DeviceId:   Resource1.DeviceId,
					Href:       Resource1.Href,
					Content:    &Resource1.Content,
					Types:      Resource1.ResourceTypes,
				},
				Resource2.Id: {
					ResourceId: Resource2.Id,
					DeviceId:   Resource2.DeviceId,
					Href:       Resource2.Href,
					Content:    &Resource2.Content,
					Types:      Resource2.ResourceTypes,
				},
			},
		},
		{
			name: "filter by device Id and type",
			args: args{
				req: &pbRS.RetrieveResourcesValuesRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					DeviceIdsFilter:      []string{Resource1.DeviceId},
					TypeFilter:           []string{Resource1.ResourceTypes[0]},
				},
			},
			want: map[string]*pbRS.ResourceValue{
				Resource1.Id: {
					ResourceId: Resource1.Id,
					DeviceId:   Resource1.DeviceId,
					Href:       Resource1.Href,
					Content:    &Resource1.Content,
					Types:      Resource1.ResourceTypes,
				},
			},
		},
		{
			name: "list all resources of user",
			args: args{
				req: &pbRS.RetrieveResourcesValuesRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
				},
			},
			want: map[string]*pbRS.ResourceValue{
				Resource1.Id: {
					ResourceId: Resource1.Id,
					DeviceId:   Resource1.DeviceId,
					Href:       Resource1.Href,
					Content:    &Resource1.Content,
					Types:      Resource1.ResourceTypes,
				},
				Resource2.Id: {
					ResourceId: Resource2.Id,
					DeviceId:   Resource2.DeviceId,
					Href:       Resource2.Href,
					Content:    &Resource2.Content,
					Types:      Resource2.ResourceTypes,
				},
				Resource3.Id: {
					ResourceId: Resource3.Id,
					DeviceId:   Resource3.DeviceId,
					Href:       Resource3.Href,
					Content:    &Resource3.Content,
					Types:      Resource3.ResourceTypes,
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
	resourceSubscriber, err := nats.NewSubscriber(natsCfg, pool.Submit, func(err error) { require.NoError(t, err) }, nats.WithTLS(tlsConfig))
	require.NoError(t, err)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")
	resourceProjection, err := NewProjection(ctx, "test", testCreateEventstore(), resourceSubscriber, time.Second)
	require.NoError(t, err)
	r := NewRequestHandler(mockAuthorizationServiceClient{}, resourceProjection)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newMockRetrieveResourcesValues(ctx)
			err := r.RetrieveResourcesValues(tt.args.req, srv)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, srv.resourceValues)
		})
	}
}

func TestRequestHandler_GetResourceLinks(t *testing.T) {
	type args struct {
		request pbRD.GetResourceLinksRequest
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]*pbRD.ResourceLink
		wantErr bool
	}{
		{
			name: "list unauthorized device",
			args: args{
				request: pbRD.GetResourceLinksRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					DeviceIdsFilter:      []string{Resource0.DeviceId},
				},
			},
			wantErr: true,
		},
		{
			name: "list one device - filter by device Id",
			args: args{
				request: pbRD.GetResourceLinksRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					DeviceIdsFilter:      []string{Resource1.DeviceId},
				},
			},
			want: map[string]*pbRD.ResourceLink{
				Resource1.Id: {
					Resource: &Resource1.Resource,
				},
				Resource3.Id: {
					Resource: &Resource3.Resource,
				},
			},
		},
		{
			name: "list all devices",
			args: args{
				request: pbRD.GetResourceLinksRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
				},
			},
			want: map[string]*pbRD.ResourceLink{
				Resource1.Id: {
					Resource: &Resource1.Resource,
				},
				Resource3.Id: {
					Resource: &Resource3.Resource,
				},
				Resource2.Id: {
					Resource: &Resource2.Resource,
				},
			},
		},
		{
			name: "list one device - filter by resource type",
			args: args{
				request: pbRD.GetResourceLinksRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					TypeFilter:           []string{Resource2.ResourceTypes[0]},
				},
			},
			want: map[string]*pbRD.ResourceLink{
				Resource1.Id: {
					Resource: &Resource1.Resource,
				},
				Resource2.Id: {
					Resource: &Resource2.Resource,
				},
			},
		},
		{
			name: "list one device - combination",
			args: args{
				request: pbRD.GetResourceLinksRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					DeviceIdsFilter:      []string{Resource1.Resource.DeviceId},
					TypeFilter:           []string{Resource3.Resource.ResourceTypes[1]},
				},
			},
			want: map[string]*pbRD.ResourceLink{
				Resource3.Id: {
					Resource: &Resource3.Resource,
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
	resourceSubscriber, err := nats.NewSubscriber(natsCfg, pool.Submit, func(err error) { require.NoError(t, err) }, nats.WithTLS(tlsConfig))
	require.NoError(t, err)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")
	resourceProjection, err := NewProjection(ctx, "test", testCreateEventstore(), resourceSubscriber, time.Second)
	require.NoError(t, err)
	r := NewRequestHandler(mockAuthorizationServiceClient{}, resourceProjection)

	for _, tt := range tests {
		fn := func(t *testing.T) {
			got := newMockResourceDirectoryGetResourceLinks(ctx)
			err := r.GetResourceLinks(&tt.args.request, got)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got.resourceLinks)
		}
		t.Run(tt.name, fn)
	}
}

func TestRequestHandler_GetDevices(t *testing.T) {
	type args struct {
		request pbDD.GetDevicesRequest
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]*pbDD.Device
		wantErr bool
	}{
		{
			name: "list ONLINE devices",
			args: args{
				request: pbDD.GetDevicesRequest{
					StatusFilter: []pbDD.Status{pbDD.Status_ONLINE},
				},
			},
			want: map[string]*pbDD.Device{
				ddResource2.Resource.DeviceId: {
					Id:       Resource2.Resource.DeviceId,
					Resource: testMakeDeviceResouceProtobuf(Resource2.Resource.DeviceId, deviceResourceTypes),
					IsOnline: true,
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
	resourceSubscriber, err := nats.NewSubscriber(natsCfg, pool.Submit, func(err error) { require.NoError(t, err) }, nats.WithTLS(tlsConfig))
	require.NoError(t, err)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")
	resourceProjection, err := NewProjection(ctx, "test", testCreateResourceDeviceEventstores(), resourceSubscriber, time.Second)
	require.NoError(t, err)
	r := NewRequestHandler(mockAuthorizationServiceClient{}, resourceProjection)

	for _, rr := range tests {
		fn := func(t *testing.T) {
			got := newMockDeviceDirectoryGetDevicesServer(ctx)
			err := r.GetDevices(&rr.args.request, got)
			if rr.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, rr.want, got.devices)
		}
		t.Run(rr.name, fn)
	}
}

type ResourceContent struct {
	pbRA.Resource
	pbRA.Content
}

var Resource0 = ResourceContent{
	pbRA.Resource{Id: "a", DeviceId: "0", ResourceTypes: []string{"t0"}},
	pbRA.Content{Data: []byte("0.a")},
}

var Resource1 = ResourceContent{
	pbRA.Resource{Id: "b", DeviceId: "1", ResourceTypes: []string{"t1", "t2"}},
	pbRA.Content{Data: []byte("1.b")},
}

var Resource2 = ResourceContent{
	pbRA.Resource{Id: "c", DeviceId: "2", ResourceTypes: []string{"t1"}},
	pbRA.Content{Data: []byte("2.c")},
}

var Resource3 = ResourceContent{
	pbRA.Resource{Id: "d", DeviceId: "1", ResourceTypes: []string{"t3", "t8"}},
	pbRA.Content{Data: []byte("1.d")},
}

func testCreateEventstore() *mockEventStore.MockEventStore {
	store := mockEventStore.NewMockEventStore()
	store.Append(Resource0.DeviceId, Resource0.Id, mockEvents.MakeResourcePublishedEvent(Resource0.Resource, cqrs.MakeEventMeta("a", 0, 0)))
	store.Append(Resource0.DeviceId, Resource0.Id, mockEvents.MakeResourceChangedEvent(Resource0.Id, Resource0.DeviceId, Resource0.Content, cqrs.MakeEventMeta("a", 0, 1)))
	store.Append(Resource1.DeviceId, Resource1.Id, mockEvents.MakeResourcePublishedEvent(Resource1.Resource, cqrs.MakeEventMeta("a", 0, 0)))
	store.Append(Resource1.DeviceId, Resource1.Id, mockEvents.MakeResourceChangedEvent(Resource1.Id, Resource1.DeviceId, Resource1.Content, cqrs.MakeEventMeta("a", 0, 1)))
	store.Append(Resource2.DeviceId, Resource2.Id, mockEvents.MakeResourcePublishedEvent(Resource2.Resource, cqrs.MakeEventMeta("a", 0, 0)))
	store.Append(Resource2.DeviceId, Resource2.Id, mockEvents.MakeResourceChangedEvent(Resource2.Id, Resource2.DeviceId, Resource2.Content, cqrs.MakeEventMeta("a", 0, 1)))
	store.Append(Resource3.DeviceId, Resource3.Id, mockEvents.MakeResourcePublishedEvent(Resource3.Resource, cqrs.MakeEventMeta("a", 0, 0)))
	store.Append(Resource3.DeviceId, Resource3.Id, mockEvents.MakeResourceChangedEvent(Resource3.Id, Resource3.DeviceId, Resource3.Content, cqrs.MakeEventMeta("a", 0, 1)))
	return store
}

type mockRetrieveResourcesValues struct {
	resourceValues map[string]*pbRS.ResourceValue
	ctx            context.Context
	grpc.ServerStream
}

func newMockRetrieveResourcesValues(ctx context.Context) *mockRetrieveResourcesValues {
	return &mockRetrieveResourcesValues{
		ctx: ctx,
	}
}

func (d *mockRetrieveResourcesValues) Send(r *pbRS.ResourceValue) error {
	if d.resourceValues == nil {
		d.resourceValues = make(map[string]*pbRS.ResourceValue)
	}
	d.resourceValues[r.ResourceId] = r
	return nil
}

func (d *mockRetrieveResourcesValues) Context() context.Context {
	return d.ctx
}

type mockAuthorizationServiceClient struct {
	pbAS.AuthorizationServiceClient
}

type mockGetUserDevicesClientClient struct {
	resourceLink []*pbAS.UserDevice
	i            int
	grpc.ClientStream
}

func (r *mockGetUserDevicesClientClient) Recv() (*pbAS.UserDevice, error) {
	if r.i >= len(r.resourceLink) {
		return nil, io.EOF
	}
	res := r.resourceLink[r.i]
	r.i++
	return res, nil
}

func (c mockAuthorizationServiceClient) GetUserDevices(ctx context.Context, in *pbAS.GetUserDevicesRequest, opts ...grpc.CallOption) (pbAS.AuthorizationService_GetUserDevicesClient, error) {
	deviceIds := []string{ /*Resource0.DeviceId,*/ Resource1.DeviceId, Resource2.DeviceId}
	userDevices := make([]*pbAS.UserDevice, 0, 16)
	for _, d := range deviceIds {
		if len(in.UserIdsFilter) == 0 {
			userDevices = append(userDevices, &pbAS.UserDevice{DeviceId: d, UserId: ""})
		} else {
			for _, userID := range in.UserIdsFilter {
				userDevices = append(userDevices, &pbAS.UserDevice{DeviceId: d, UserId: userID})
			}
		}
	}
	return &mockGetUserDevicesClientClient{
		resourceLink: userDevices,
	}, nil
}

type mockResourceDirectoryGetResourceLinks struct {
	resourceLinks map[string]*pbRD.ResourceLink
	ctx           context.Context
	grpc.ServerStream
}

func newMockResourceDirectoryGetResourceLinks(ctx context.Context) *mockResourceDirectoryGetResourceLinks {
	return &mockResourceDirectoryGetResourceLinks{
		ctx: ctx,
	}
}

func (d *mockResourceDirectoryGetResourceLinks) Send(link *pbRD.ResourceLink) error {
	if d.resourceLinks == nil {
		d.resourceLinks = make(map[string]*pbRD.ResourceLink)
	}
	d.resourceLinks[link.Resource.Id] = link
	return nil
}

func (d *mockResourceDirectoryGetResourceLinks) Context() context.Context {
	return d.ctx
}

type mockDeviceDirectoryGetDevicesServer struct {
	devices map[string]*pbDD.Device
	ctx     context.Context
	grpc.ServerStream
}

func newMockDeviceDirectoryGetDevicesServer(ctx context.Context) *mockDeviceDirectoryGetDevicesServer {
	return &mockDeviceDirectoryGetDevicesServer{
		ctx: ctx,
	}
}

func (d *mockDeviceDirectoryGetDevicesServer) Send(device *pbDD.Device) error {
	if d.devices == nil {
		d.devices = make(map[string]*pbDD.Device)
	}
	d.devices[device.Id] = device
	return nil
}

func (d *mockDeviceDirectoryGetDevicesServer) Context() context.Context {
	return d.ctx
}
