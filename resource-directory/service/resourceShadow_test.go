package service_test

import (
	"context"
	"fmt"
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
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/resource-directory/service"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
)

func TestResourceTwinGetResources(t *testing.T) {
	type args struct {
		req *pb.GetResourcesRequest
	}
	tests := []struct {
		name string
		args args
		want map[string]*pb.Resource
	}{
		{
			name: "list unauthorized device",
			args: args{
				req: &pb.GetResourcesRequest{
					DeviceIdFilter: []string{Resource0.DeviceId},
				},
			},
		},

		{
			name: "filter by resource Id",
			args: args{
				req: &pb.GetResourcesRequest{
					ResourceIdFilter: []*pb.ResourceIdFilter{
						{
							ResourceId: Resource1.GetResourceID(),
						},
						{
							ResourceId: Resource2.GetResourceID(),
						},
					},
				},
			},
			want: map[string]*pb.Resource{
				Resource1.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource1.DeviceId,
							Href:     Resource1.Href,
						},
						Content:       Resource1.Content,
						ResourceTypes: Resource1.ResourceTypes,
					},
					Types: Resource1.ResourceTypes,
				},
				Resource2.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource2.DeviceId,
							Href:     Resource2.Href,
						},
						Content:       Resource2.Content,
						ResourceTypes: Resource2.ResourceTypes,
					},
					Types: Resource2.ResourceTypes,
				},
			},
		},

		{
			name: "filter by device Id",
			args: args{
				req: &pb.GetResourcesRequest{
					DeviceIdFilter: []string{Resource1.DeviceId},
				},
			},
			want: map[string]*pb.Resource{
				Resource1.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource1.DeviceId,
							Href:     Resource1.Href,
						},
						Content:       Resource1.Content,
						ResourceTypes: Resource1.ResourceTypes,
					},
					Types: Resource1.ResourceTypes,
				},
				Resource3.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource3.DeviceId,
							Href:     Resource3.Href,
						},
						Content:       Resource3.Content,
						ResourceTypes: Resource3.ResourceTypes,
					},
					Types: Resource3.ResourceTypes,
				},
			},
		},

		{
			name: "filter by type",
			args: args{
				req: &pb.GetResourcesRequest{
					TypeFilter: []string{Resource2.ResourceTypes[0]},
				},
			},
			want: map[string]*pb.Resource{
				Resource1.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource1.DeviceId,
							Href:     Resource1.Href,
						},
						Content:       Resource1.Content,
						ResourceTypes: Resource1.ResourceTypes,
					},
					Types: Resource1.ResourceTypes,
				},
				Resource2.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource2.DeviceId,
							Href:     Resource2.Href,
						},
						Content:       Resource2.Content,
						ResourceTypes: Resource2.ResourceTypes,
					},
					Types: Resource2.ResourceTypes,
				},
			},
		},

		{
			name: "filter by device ID and type",
			args: args{
				req: &pb.GetResourcesRequest{
					DeviceIdFilter: []string{Resource1.DeviceId},
					TypeFilter:     []string{Resource1.ResourceTypes[0]},
				},
			},
			want: map[string]*pb.Resource{
				Resource1.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource1.DeviceId,
							Href:     Resource1.Href,
						},
						Content:       Resource1.Content,
						ResourceTypes: Resource1.ResourceTypes,
					},
					Types: Resource1.ResourceTypes,
				},
			},
		},

		{
			name: "list all resources of user",
			args: args{
				req: &pb.GetResourcesRequest{},
			},
			want: map[string]*pb.Resource{
				Resource1.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource1.DeviceId,
							Href:     Resource1.Href,
						},
						Content:       Resource1.Content,
						ResourceTypes: Resource1.ResourceTypes,
					},
					Types: Resource1.ResourceTypes,
				},
				Resource2.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource2.DeviceId,
							Href:     Resource2.Href,
						},
						Content:       Resource2.Content,
						ResourceTypes: Resource2.ResourceTypes,
					},
					Types: Resource2.ResourceTypes,
				},
				Resource3.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource3.DeviceId,
							Href:     Resource3.Href,
						},
						Content:       Resource3.Content,
						ResourceTypes: Resource3.ResourceTypes,
					},
					Types: Resource3.ResourceTypes,
				},
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
	resourceProjection, err := service.NewProjection(ctx, "test", testCreateEventstore(), resourceSubscriber, mf, time.Second)
	require.NoError(t, err)

	rd := service.NewResourceTwin(resourceProjection, []string{ /*Resource0.DeviceId,*/ Resource1.DeviceId, Resource2.DeviceId})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println(tt.name)
			var s testGrpcGateway_GetResourcesServer
			err := rd.GetResources(tt.args.req, &s)
			require.NoError(t, err)
			test.CheckProtobufs(t, tt.want, s.got, test.RequireToCheckFunc(require.Equal))
		})
	}
}

type testGrpcGateway_GetResourcesServer struct {
	got map[string]*pb.Resource
	grpc.ServerStream
}

func (s *testGrpcGateway_GetResourcesServer) Context() context.Context {
	return context.Background()
}

func (s *testGrpcGateway_GetResourcesServer) Send(d *pb.Resource) error {
	if s.got == nil {
		s.got = make(map[string]*pb.Resource)
	}
	d.Data.AuditContext = nil
	d.Data.EventMetadata = nil
	s.got[d.GetData().GetResourceId().GetHref()] = d
	return nil
}
