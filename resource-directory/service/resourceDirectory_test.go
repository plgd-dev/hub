package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
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
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestResourceDirectoryGetResourceLinks(t *testing.T) {
	type args struct {
		request *pb.GetResourceLinksRequest
	}
	tests := []struct {
		name string
		args args
		want map[string]*events.ResourceLinksPublished
	}{
		{
			name: "list one device - filter by device Id",
			args: args{
				request: &pb.GetResourceLinksRequest{
					DeviceIdFilter: []string{Resource1.DeviceId},
				},
			},
			want: map[string]*events.ResourceLinksPublished{
				Resource1.DeviceId: {
					DeviceId: Resource1.DeviceId,
					Resources: []*commands.Resource{
						Resource1.Resource,
						Resource3.Resource,
					},
					AuditContext: commands.NewAuditContext("userId", ""),
				},
			},
		},
	}
	logger := log.NewLogger(log.MakeDefaultConfig())
	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() {
		err := fileWatcher.Close()
		require.NoError(t, err)
	}()

	pool, err := ants.NewPool(1)
	require.NoError(t, err)
	naClient, resourceSubscriber, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), fileWatcher,
		logger,
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
	resourceProjection, err := service.NewProjection(ctx, "test", testCreateEventstore(), resourceSubscriber, mf, time.Second)
	require.NoError(t, err)

	rd := service.NewResourceDirectory(resourceProjection, []string{ /*Resource0.DeviceId,*/ Resource1.DeviceId, Resource2.DeviceId})

	for _, tt := range tests {
		fn := func(t *testing.T) {
			var s testGrpcGateway_GetResourceLinksServer
			err := rd.GetResourceLinks(tt.args.request, &s)
			require.NoError(t, err)
			test.CheckProtobufs(t, tt.want, s.got, test.AssertToCheckFunc(assert.Equal))
		}
		t.Run(tt.name, fn)
	}
}

func newResourceContent(deviceID, href string, resourceTypesp []string, content *commands.Content) *ResourceContent {
	return &ResourceContent{
		Resource: &commands.Resource{Href: href, DeviceId: deviceID, ResourceTypes: resourceTypesp},
		Content:  content,
	}
}

var (
	Resource0 = newResourceContent("0", "/a", []string{"t0"}, &commands.Content{Data: []byte("0.a")})
	Resource1 = newResourceContent("1", "/b", []string{"t1", "t2"}, &commands.Content{Data: []byte("1.b")})
	Resource2 = newResourceContent("2", "/c", []string{"t1"}, &commands.Content{Data: []byte("2.c")})
	Resource3 = newResourceContent("1", "/d", []string{"t3", "t8"}, &commands.Content{Data: []byte("1.d")})
)

func testCreateEventstore() *mockEvents.MockEventStore {
	store := mockEvents.NewMockEventStore()
	store.Append(Resource0.DeviceId, commands.MakeLinksResourceUUID(Resource0.DeviceId), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{Resource0.Resource}, Resource0.GetDeviceId(), events.MakeEventMeta("a", 0, 0)))
	store.Append(Resource1.DeviceId, commands.MakeLinksResourceUUID(Resource1.DeviceId), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{Resource1.Resource}, Resource1.GetDeviceId(), events.MakeEventMeta("a", 0, 0)))
	store.Append(Resource2.DeviceId, commands.MakeLinksResourceUUID(Resource2.DeviceId), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{Resource2.Resource}, Resource2.GetDeviceId(), events.MakeEventMeta("a", 0, 0)))
	store.Append(Resource3.DeviceId, commands.MakeLinksResourceUUID(Resource3.DeviceId), mockEvents.MakeResourceLinksPublishedEvent([]*commands.Resource{Resource3.Resource}, Resource3.GetDeviceId(), events.MakeEventMeta("a", 0, 0)))
	store.Append(Resource0.DeviceId, Resource0.ToUUID(), mockEvents.MakeResourceChangedEvent(Resource0.Resource.GetResourceID(), Resource0.Content, events.MakeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	store.Append(Resource1.DeviceId, Resource1.ToUUID(), mockEvents.MakeResourceChangedEvent(Resource1.Resource.GetResourceID(), Resource1.Content, events.MakeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	store.Append(Resource2.DeviceId, Resource2.ToUUID(), mockEvents.MakeResourceChangedEvent(Resource2.Resource.GetResourceID(), Resource2.Content, events.MakeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	store.Append(Resource3.DeviceId, Resource3.ToUUID(), mockEvents.MakeResourceChangedEvent(Resource3.Resource.GetResourceID(), Resource3.Content, events.MakeEventMeta("a", 0, 0), mockEvents.MakeAuditContext("userId", "0")))
	return store
}

type testGrpcGateway_GetResourceLinksServer struct {
	got map[string]*events.ResourceLinksPublished
	grpc.ServerStream
}

func (s *testGrpcGateway_GetResourceLinksServer) Context() context.Context {
	return context.Background()
}

func (s *testGrpcGateway_GetResourceLinksServer) Send(d *events.ResourceLinksPublished) error {
	if s.got == nil {
		s.got = make(map[string]*events.ResourceLinksPublished)
	}
	s.got[d.DeviceId] = pbTest.CleanUpResourceLinksPublished(d, true)
	return nil
}
