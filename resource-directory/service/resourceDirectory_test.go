package service

import (
	"context"
	"testing"
	"time"

	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/security/certManager"
	"github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs/eventbus/nats"
	pbCQRS "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
	pbRD "github.com/go-ocf/ocf-cloud/resource-directory/pb/resource-directory"
	"github.com/kelseyhightower/envconfig"
	"github.com/panjf2000/ants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestResourceDirectory_GetResourceLinks(t *testing.T) {
	type args struct {
		request pbRD.GetResourceLinksRequest
	}
	test := []struct {
		name     string
		args     args
		want     map[string]*pbRD.ResourceLink
		wantCode codes.Code
		wantErr  bool
	}{
		{
			name: "list one device - filter by device Id",
			args: args{
				request: pbRD.GetResourceLinksRequest{
					AuthorizationContext: &pbCQRS.AuthorizationContext{},
					DeviceIdsFilter:      []string{Resource1.DeviceId},
				},
			},
			want: map[string]*pbRD.ResourceLink{
				Resource1.Id: &pbRD.ResourceLink{
					Resource: &Resource1.Resource,
				},
				Resource3.Id: &pbRD.ResourceLink{
					&Resource3.Resource,
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

	rd := NewResourceDirectory(resourceProjection, []string{ /*Resource0.DeviceId,*/ Resource1.DeviceId, Resource2.DeviceId})

	for _, tt := range test {
		fn := func(t *testing.T) {
			var got map[string]*pbRD.ResourceLink
			statusCode, err := rd.GetResourceLinks(ctx, &tt.args.request, func(resourceLink *pbRD.ResourceLink) error {
				if got == nil {
					got = make(map[string]*pbRD.ResourceLink)
				}
				got[resourceLink.Resource.Id] = resourceLink
				return nil
			})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantCode, statusCode)
			assert.Equal(t, tt.want, got)
		}
		t.Run(tt.name, fn)
	}
}
