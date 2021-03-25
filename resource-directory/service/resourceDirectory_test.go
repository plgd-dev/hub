package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	mockEventStore "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	mockEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils/notification"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/cloud/resource-directory/service"
	"github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/kit/security/certManager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestResourceDirectory_GetResourceLinks(t *testing.T) {
	linkToPtr := func(l pb.ResourceLink) *pb.ResourceLink {
		return &l
	}
	type args struct {
		request pb.GetResourceLinksRequest
	}
	tests := []struct {
		name     string
		args     args
		want     map[string]*pb.ResourceLink
		wantCode codes.Code
		wantErr  bool
	}{
		{
			name: "list one device - filter by device Id",
			args: args{
				request: pb.GetResourceLinksRequest{
					DeviceIdsFilter: []string{Resource1.DeviceId},
				},
			},
			want: map[string]*pb.ResourceLink{
				utils.MakeResourceId(Resource1.DeviceId, Resource1.Href): linkToPtr(pb.RAResourceToProto(&Resource1.Resource)),
				utils.MakeResourceId(Resource3.DeviceId, Resource3.Href): linkToPtr(pb.RAResourceToProto(&Resource3.Resource)),
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

	resourceProjection, err := service.NewProjection(ctx, "test", testCreateEventstore(), resourceSubscriber, service.NewResourceCtx(subscriptions, updateNotificationContainer, retrieveNotificationContainer, deleteNotificationContainer), time.Second)
	require.NoError(t, err)

	rd := service.New(resourceProjection, []string{ /*Resource0.DeviceId,*/ Resource1.DeviceId, Resource2.DeviceId})

	for _, tt := range tests {
		fn := func(t *testing.T) {
			var s testGrpcGateway_GetResourceLinksServer
			err := rd.GetResourceLinks(&tt.args.request, &s)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantCode, status.Convert(err).Code())
			test.CheckProtobufs(t, tt.want, s.got, test.AssertToCheckFunc(assert.Equal))
		}
		t.Run(tt.name, fn)
	}
}

func newResourceContent(deviceID, href string, resourceTypesp []string, content pbRA.Content) ResourceContent {
	return ResourceContent{
		Resource: pbRA.Resource{Id: utils.MakeResourceId(deviceID, href), Href: href, DeviceId: deviceID, ResourceTypes: resourceTypesp},
		Content:  content,
	}
}

var Resource0 = newResourceContent("0", "a", []string{"t0"}, pbRA.Content{Data: []byte("0.a")})
var Resource1 = newResourceContent("1", "b", []string{"t1", "t2"}, pbRA.Content{Data: []byte("1.b")})
var Resource2 = newResourceContent("2", "c", []string{"t1"}, pbRA.Content{Data: []byte("2.c")})
var Resource3 = newResourceContent("1", "d", []string{"t3", "t8"}, pbRA.Content{Data: []byte("1.d")})

func testCreateEventstore() *mockEventStore.MockEventStore {
	store := mockEventStore.NewMockEventStore()
	store.Append(Resource0.DeviceId, Resource0.Id, mockEvents.MakeResourcePublishedEvent(Resource0.Resource, utils.MakeEventMeta("a", 0, 0)))
	store.Append(Resource0.DeviceId, Resource0.Id, mockEvents.MakeResourceChangedEvent(Resource0.Id, Resource0.DeviceId, Resource0.Content, utils.MakeEventMeta("a", 0, 1)))
	store.Append(Resource1.DeviceId, Resource1.Id, mockEvents.MakeResourcePublishedEvent(Resource1.Resource, utils.MakeEventMeta("a", 0, 0)))
	store.Append(Resource1.DeviceId, Resource1.Id, mockEvents.MakeResourceChangedEvent(Resource1.Id, Resource1.DeviceId, Resource1.Content, utils.MakeEventMeta("a", 0, 1)))
	store.Append(Resource2.DeviceId, Resource2.Id, mockEvents.MakeResourcePublishedEvent(Resource2.Resource, utils.MakeEventMeta("a", 0, 0)))
	store.Append(Resource2.DeviceId, Resource2.Id, mockEvents.MakeResourceChangedEvent(Resource2.Id, Resource2.DeviceId, Resource2.Content, utils.MakeEventMeta("a", 0, 1)))
	store.Append(Resource3.DeviceId, Resource3.Id, mockEvents.MakeResourcePublishedEvent(Resource3.Resource, utils.MakeEventMeta("a", 0, 0)))
	store.Append(Resource3.DeviceId, Resource3.Id, mockEvents.MakeResourceChangedEvent(Resource3.Id, Resource3.DeviceId, Resource3.Content, utils.MakeEventMeta("a", 0, 1)))
	return store
}

type testGrpcGateway_GetResourceLinksServer struct {
	got map[string]*pb.ResourceLink
	grpc.ServerStream
}

func (s *testGrpcGateway_GetResourceLinksServer) Context() context.Context {
	return context.Background()
}

func (s *testGrpcGateway_GetResourceLinksServer) Send(d *pb.ResourceLink) error {
	if s.got == nil {
		s.got = make(map[string]*pb.ResourceLink)
	}
	s.got[utils.MakeResourceId(d.DeviceId, d.Href)] = d
	return nil
}
