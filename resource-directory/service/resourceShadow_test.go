package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
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

func TestResourceShadow_GetResources(t *testing.T) {
	type args struct {
		req *pb.GetResourcesRequest
	}
	tests := []struct {
		name           string
		args           args
		wantStatusCode codes.Code
		wantErr        bool
		want           map[string]*pb.Resource
	}{

		{
			name: "list unauthorized device",
			args: args{
				req: &pb.GetResourcesRequest{
					DeviceIdsFilter: []string{Resource0.DeviceId},
				},
			},
			wantStatusCode: codes.NotFound,
			wantErr:        true,
		},

		{
			name: "filter by resource Id",
			args: args{
				req: &pb.GetResourcesRequest{
					ResourceIdsFilter: []string{
						Resource1.ToResourceIDString(),
						Resource2.ToResourceIDString(),
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
						Content: &Resource1.Content,
					},
					Types: Resource1.ResourceTypes,
				},
				Resource2.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource2.DeviceId,
							Href:     Resource2.Href,
						},
						Content: &Resource2.Content,
					},
					Types: Resource2.ResourceTypes,
				},
			},
		},

		{
			name: "filter by device Id",
			args: args{
				req: &pb.GetResourcesRequest{
					DeviceIdsFilter: []string{Resource1.DeviceId},
				},
			},
			want: map[string]*pb.Resource{
				Resource1.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource1.DeviceId,
							Href:     Resource1.Href,
						},
						Content: &Resource1.Content,
					},
					Types: Resource1.ResourceTypes,
				},
				Resource3.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource3.DeviceId,
							Href:     Resource3.Href,
						},
						Content: &Resource3.Content,
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
						Content: &Resource1.Content,
					},
					Types: Resource1.ResourceTypes,
				},
				Resource2.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource2.DeviceId,
							Href:     Resource2.Href,
						},
						Content: &Resource2.Content,
					},
					Types: Resource2.ResourceTypes,
				},
			},
		},

		{
			name: "filter by device Id and type",
			args: args{
				req: &pb.GetResourcesRequest{
					DeviceIdsFilter: []string{Resource1.DeviceId},
					TypeFilter:      []string{Resource1.ResourceTypes[0]},
				},
			},
			want: map[string]*pb.Resource{
				Resource1.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource1.DeviceId,
							Href:     Resource1.Href,
						},
						Content: &Resource1.Content,
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
						Content: &Resource1.Content,
					},
					Types: Resource1.ResourceTypes,
				},
				Resource2.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource2.DeviceId,
							Href:     Resource2.Href,
						},
						Content: &Resource2.Content,
					},
					Types: Resource2.ResourceTypes,
				},
				Resource3.Href: {
					Data: &events.ResourceChanged{
						ResourceId: &commands.ResourceId{
							DeviceId: Resource3.DeviceId,
							Href:     Resource3.Href,
						},
						Content: &Resource3.Content,
					},
					Types: Resource3.ResourceTypes,
				},
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

	rd := service.NewResourceShadow(resourceProjection, []string{ /*Resource0.DeviceId,*/ Resource1.DeviceId, Resource2.DeviceId})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println(tt.name)
			var s testGrpcGateway_GetResourcesServer
			err := rd.GetResources(tt.args.req, &s)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantStatusCode, status.Convert(err).Code())
			test.CheckProtobufs(t, tt.want, s.got, test.AssertToCheckFunc(assert.Equal))
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
