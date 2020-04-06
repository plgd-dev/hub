package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/security/certManager"
	"github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs/eventbus/nats"
	pbCQRS "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
	pbRS "github.com/go-ocf/ocf-cloud/resource-directory/pb/resource-shadow"
	"github.com/kelseyhightower/envconfig"
	"github.com/panjf2000/ants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestResourceShadow_RetrieveResourcesValues(t *testing.T) {
	type args struct {
		req *pbRS.RetrieveResourcesValuesRequest
	}
	tests := []struct {
		name           string
		args           args
		wantStatusCode codes.Code
		wantErr        bool
		want           map[string]*pbRS.ResourceValue
	}{

		{
			name: "list unauthorized device",
			args: args{
				req: &pbRS.RetrieveResourcesValuesRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					DeviceIdsFilter:      []string{Resource0.DeviceId},
				},
			},
			wantStatusCode: codes.NotFound,
			wantErr:        true,
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
				Resource1.Id: &pbRS.ResourceValue{
					ResourceId: Resource1.Id,
					DeviceId:   Resource1.DeviceId,
					Href:       Resource1.Href,
					Content:    &Resource1.Content,
					Types:      Resource1.ResourceTypes,
				},
				Resource2.Id: &pbRS.ResourceValue{
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
				Resource1.Id: &pbRS.ResourceValue{
					ResourceId: Resource1.Id,
					DeviceId:   Resource1.DeviceId,
					Href:       Resource1.Href,
					Content:    &Resource1.Content,
					Types:      Resource1.ResourceTypes,
				},
				Resource3.Id: &pbRS.ResourceValue{
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
				Resource1.Id: &pbRS.ResourceValue{
					ResourceId: Resource1.Id,
					DeviceId:   Resource1.DeviceId,
					Href:       Resource1.Href,
					Content:    &Resource1.Content,
					Types:      Resource1.ResourceTypes,
				},
				Resource2.Id: &pbRS.ResourceValue{
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
				Resource1.Id: &pbRS.ResourceValue{
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
				Resource1.Id: &pbRS.ResourceValue{
					ResourceId: Resource1.Id,
					DeviceId:   Resource1.DeviceId,
					Href:       Resource1.Href,
					Content:    &Resource1.Content,
					Types:      Resource1.ResourceTypes,
				},
				Resource2.Id: &pbRS.ResourceValue{
					ResourceId: Resource2.Id,
					DeviceId:   Resource2.DeviceId,
					Href:       Resource2.Href,
					Content:    &Resource2.Content,
					Types:      Resource2.ResourceTypes,
				},
				Resource3.Id: &pbRS.ResourceValue{
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
	resourceSubscriber, err := nats.NewSubscriber(natsCfg, pool.Submit, func(err error) { require.NoError(t, err) }, nats.WithTLS(&tlsConfig))
	require.NoError(t, err)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")
	resourceProjection, err := NewProjection(ctx, "test", testCreateEventstore(), resourceSubscriber, time.Second)
	require.NoError(t, err)

	rd := NewResourceShadow(resourceProjection, []string{ /*Resource0.DeviceId,*/ Resource1.DeviceId, Resource2.DeviceId})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println(tt.name)
			var got map[string]*pbRS.ResourceValue
			gotStatusCode, err := rd.RetrieveResourcesValues(context.Background(), tt.args.req, func(r *pbRS.ResourceValue) error {
				if got == nil {
					got = make(map[string]*pbRS.ResourceValue)
				}
				got[r.ResourceId] = r
				return nil
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantStatusCode, gotStatusCode)
			assert.Equal(t, tt.want, got)
			got = nil
		})
	}
}
