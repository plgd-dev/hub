package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/cloud/grpc-gateway/service"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/notification"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/security/certManager"
	"github.com/kelseyhightower/envconfig"
	"github.com/panjf2000/ants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestResourceShadow_RetrieveResourcesValues(t *testing.T) {
	type args struct {
		req *pb.RetrieveResourcesValuesRequest
	}
	tests := []struct {
		name           string
		args           args
		wantStatusCode codes.Code
		wantErr        bool
		want           map[string]*pb.ResourceValue
	}{

		{
			name: "list unauthorized device",
			args: args{
				req: &pb.RetrieveResourcesValuesRequest{
					DeviceIdsFilter: []string{Resource0.DeviceId},
				},
			},
			wantStatusCode: codes.NotFound,
			wantErr:        true,
		},

		{
			name: "filter by resource Id",
			args: args{
				req: &pb.RetrieveResourcesValuesRequest{
					ResourceIdsFilter: []*pb.ResourceId{
						{
							DeviceId:         Resource1.DeviceId,
							ResourceLinkHref: Resource1.Href,
						}, {
							DeviceId:         Resource2.DeviceId,
							ResourceLinkHref: Resource2.Href,
						},
					},
				},
			},
			want: map[string]*pb.ResourceValue{
				Resource1.Id: {
					ResourceId: &pb.ResourceId{
						DeviceId:         Resource1.DeviceId,
						ResourceLinkHref: Resource1.Href,
					},
					Content: pb.RAContent2Content(&Resource1.Content),
					Types:   Resource1.ResourceTypes,
				},
				Resource2.Id: {
					ResourceId: &pb.ResourceId{
						DeviceId:         Resource2.DeviceId,
						ResourceLinkHref: Resource2.Href,
					},
					Content: pb.RAContent2Content(&Resource2.Content),
					Types:   Resource2.ResourceTypes,
				},
			},
		},

		{
			name: "filter by device Id",
			args: args{
				req: &pb.RetrieveResourcesValuesRequest{
					DeviceIdsFilter: []string{Resource1.DeviceId},
				},
			},
			want: map[string]*pb.ResourceValue{
				Resource1.Id: {
					ResourceId: &pb.ResourceId{
						DeviceId:         Resource1.DeviceId,
						ResourceLinkHref: Resource1.Href,
					},
					Content: pb.RAContent2Content(&Resource1.Content),
					Types:   Resource1.ResourceTypes,
				},
				Resource3.Id: {
					ResourceId: &pb.ResourceId{
						DeviceId:         Resource3.DeviceId,
						ResourceLinkHref: Resource3.Href,
					},
					Content: pb.RAContent2Content(&Resource3.Content),
					Types:   Resource3.ResourceTypes,
				},
			},
		},

		{
			name: "filter by type",
			args: args{
				req: &pb.RetrieveResourcesValuesRequest{
					TypeFilter: []string{Resource2.ResourceTypes[0]},
				},
			},
			want: map[string]*pb.ResourceValue{
				Resource1.Id: {
					ResourceId: &pb.ResourceId{
						DeviceId:         Resource1.DeviceId,
						ResourceLinkHref: Resource1.Href,
					},
					Content: pb.RAContent2Content(&Resource1.Content),
					Types:   Resource1.ResourceTypes,
				},
				Resource2.Id: {
					ResourceId: &pb.ResourceId{
						DeviceId:         Resource2.DeviceId,
						ResourceLinkHref: Resource2.Href,
					},
					Content: pb.RAContent2Content(&Resource2.Content),
					Types:   Resource2.ResourceTypes,
				},
			},
		},

		{
			name: "filter by device Id and type",
			args: args{
				req: &pb.RetrieveResourcesValuesRequest{
					DeviceIdsFilter: []string{Resource1.DeviceId},
					TypeFilter:      []string{Resource1.ResourceTypes[0]},
				},
			},
			want: map[string]*pb.ResourceValue{
				Resource1.Id: {
					ResourceId: &pb.ResourceId{
						DeviceId:         Resource1.DeviceId,
						ResourceLinkHref: Resource1.Href,
					},
					Content: pb.RAContent2Content(&Resource1.Content),
					Types:   Resource1.ResourceTypes,
				},
			},
		},

		{
			name: "list all resources of user",
			args: args{
				req: &pb.RetrieveResourcesValuesRequest{},
			},
			want: map[string]*pb.ResourceValue{
				Resource1.Id: {
					ResourceId: &pb.ResourceId{
						DeviceId:         Resource1.DeviceId,
						ResourceLinkHref: Resource1.Href,
					},
					Content: pb.RAContent2Content(&Resource1.Content),
					Types:   Resource1.ResourceTypes,
				},
				Resource2.Id: {
					ResourceId: &pb.ResourceId{
						DeviceId:         Resource2.DeviceId,
						ResourceLinkHref: Resource2.Href,
					},
					Content: pb.RAContent2Content(&Resource2.Content),
					Types:   Resource2.ResourceTypes,
				},
				Resource3.Id: {
					ResourceId: &pb.ResourceId{
						DeviceId:         Resource3.DeviceId,
						ResourceLinkHref: Resource3.Href,
					},
					Content: pb.RAContent2Content(&Resource3.Content),
					Types:   Resource3.ResourceTypes,
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

	subscriptions := service.NewSubscriptions()
	updateNotificationContainer := notification.NewUpdateNotificationContainer()
	retrieveNotificationContainer := notification.NewRetrieveNotificationContainer()

	resourceProjection, err := service.NewProjection(ctx, "test", testCreateEventstore(), resourceSubscriber, service.NewResourceCtx(subscriptions, updateNotificationContainer, retrieveNotificationContainer), time.Second)
	require.NoError(t, err)

	rd := service.NewResourceShadow(resourceProjection, []string{ /*Resource0.DeviceId,*/ Resource1.DeviceId, Resource2.DeviceId})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println(tt.name)
			var s testGrpcGateway_RetrieveResourcesValuesServer
			err := rd.RetrieveResourcesValues(tt.args.req, &s)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantStatusCode, status.Convert(err).Code())
			assert.Equal(t, tt.want, s.got)
		})
	}
}

type testGrpcGateway_RetrieveResourcesValuesServer struct {
	got map[string]*pb.ResourceValue
	grpc.ServerStream
}

func (s *testGrpcGateway_RetrieveResourcesValuesServer) Context() context.Context {
	return context.Background()
}

func (s *testGrpcGateway_RetrieveResourcesValuesServer) Send(d *pb.ResourceValue) error {
	if s.got == nil {
		s.got = make(map[string]*pb.ResourceValue)
	}
	s.got[cqrs.MakeResourceId(d.GetResourceId().GetDeviceId(), d.GetResourceId().GetResourceLinkHref())] = d
	return nil
}
