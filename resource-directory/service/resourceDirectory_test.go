package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	mockEventStore "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	mockEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils/notification"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/resource-directory/service"
	"github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestResourceDirectory_GetResourceLinks(t *testing.T) {
	linkToPtr := func(l *pb.ResourceLink) *pb.ResourceLink {
		return l
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
				Resource1.ToUUID(): linkToPtr(pb.RAResourceToProto(&Resource1.Resource)),
				Resource3.ToUUID(): linkToPtr(pb.RAResourceToProto(&Resource3.Resource)),
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

	resourceProjection, err := service.NewProjection(ctx, "test", testCreateEventstore(), resourceSubscriber, service.NewEventStoreModelFactory(subscriptions, updateNotificationContainer, retrieveNotificationContainer, deleteNotificationContainer, createNotificationContainer), time.Second)
	require.NoError(t, err)

	rd := service.NewResourceDirectory(resourceProjection, []string{ /*Resource0.DeviceId,*/ Resource1.DeviceId, Resource2.DeviceId})

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

func newResourceContent(deviceID, href string, resourceTypesp []string, content commands.Content) ResourceContent {
	return ResourceContent{
		Resource: commands.Resource{Href: href, DeviceId: deviceID, ResourceTypes: resourceTypesp},
		Content:  content,
	}
}

var Resource0 = newResourceContent("0", "a", []string{"t0"}, commands.Content{Data: []byte("0.a")})
var Resource1 = newResourceContent("1", "b", []string{"t1", "t2"}, commands.Content{Data: []byte("1.b")})
var Resource2 = newResourceContent("2", "c", []string{"t1"}, commands.Content{Data: []byte("2.c")})
var Resource3 = newResourceContent("1", "d", []string{"t3", "t8"}, commands.Content{Data: []byte("1.d")})

func testCreateEventstore() *mockEventStore.MockEventStore {
	store := mockEventStore.NewMockEventStore()
	store.Append(Resource0.DeviceId, commands.MakeLinksResourceUUID(Resource0.DeviceId), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&Resource0.Resource}, Resource0.GetDeviceId(), events.MakeEventMeta("a", 0, 0)))
	store.Append(Resource1.DeviceId, commands.MakeLinksResourceUUID(Resource1.DeviceId), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&Resource1.Resource}, Resource1.GetDeviceId(), events.MakeEventMeta("a", 0, 0)))
	store.Append(Resource2.DeviceId, commands.MakeLinksResourceUUID(Resource2.DeviceId), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&Resource2.Resource}, Resource2.GetDeviceId(), events.MakeEventMeta("a", 0, 0)))
	store.Append(Resource3.DeviceId, commands.MakeLinksResourceUUID(Resource3.DeviceId), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{&Resource3.Resource}, Resource3.GetDeviceId(), events.MakeEventMeta("a", 0, 0)))
	store.Append(Resource0.DeviceId, Resource0.ToUUID(), mockEvents.MakeResourceChangedEvent(Resource0.Resource.GetResourceID(), &Resource0.Content, events.MakeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	store.Append(Resource1.DeviceId, Resource1.ToUUID(), mockEvents.MakeResourceChangedEvent(Resource1.Resource.GetResourceID(), &Resource1.Content, events.MakeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	store.Append(Resource2.DeviceId, Resource2.ToUUID(), mockEvents.MakeResourceChangedEvent(Resource2.Resource.GetResourceID(), &Resource2.Content, events.MakeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	store.Append(Resource3.DeviceId, Resource3.ToUUID(), mockEvents.MakeResourceChangedEvent(Resource3.Resource.GetResourceID(), &Resource3.Content, events.MakeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
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
	s.got[(&commands.ResourceId{DeviceId: d.DeviceId, Href: d.Href}).ToUUID()] = d
	return nil
}
