package mongodb

import (
	"context"
	"testing"

	"github.com/go-ocf/kit/security/certManager"

	"github.com/go-ocf/ocf-cloud/openapi-connector/store"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStore(ctx context.Context, t *testing.T, cfg Config) *Store {
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	defer dialCertManager.Close()
	tlsConfig := dialCertManager.GetClientTLSConfig()
	s, err := NewStore(ctx, cfg, WithTLS(&tlsConfig))
	require.NoError(t, err)
	return s
}

func TestStore_UpdateLinkedCloud(t *testing.T) {
	type args struct {
		sub store.LinkedCloud
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "not found",
			args: args{
				sub: store.LinkedCloud{
					ID:           "testIDnotFound",
					Name:         "testName",
					ClientID:     "testClientID",
					ClientSecret: "testClientSecret",
					Scopes:       []string{"testScopes"},
					Endpoint: store.Endpoint{
						AuthUrl:  "testAuthUrl",
						TokenUrl: "testTokenUrl",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				sub: store.LinkedCloud{
					ID:           "testID",
					Name:         "testNameUpdated",
					ClientID:     "testClientID",
					ClientSecret: "testClientSecret",
					Scopes:       []string{"testScopes"},
					Audience:     "testAudience",
					Endpoint: store.Endpoint{
						AuthUrl:  "testAuthUrl",
						TokenUrl: "testTokenUrl",
					},
				},
			},
		},
	}

	require := require.New(t)
	var config Config
	err := envconfig.Process("", &config)
	require.NoError(err)
	ctx := context.Background()
	s := newStore(ctx, t, config)
	require.NoError(err)
	defer s.Clear(ctx)

	assert := assert.New(t)

	err = s.InsertLinkedCloud(ctx, store.LinkedCloud{
		ID:           "testID",
		Name:         "testName",
		ClientID:     "testClientID",
		ClientSecret: "testClientSecret",
		Scopes:       []string{"testScopes"},
		Endpoint: store.Endpoint{
			AuthUrl:  "testAuthUrl",
			TokenUrl: "testTokenUrl",
		},
	})
	require.NoError(err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.UpdateLinkedCloud(ctx, tt.args.sub)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestStore_RemoveLinkedCloud(t *testing.T) {
	type args struct {
		LinkedCloudId string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "not found",
			args: args{
				LinkedCloudId: "notFound",
			},
			wantErr: true,
		},

		{
			name: "valid",
			args: args{
				LinkedCloudId: "testID",
			},
		},
	}

	require := require.New(t)
	var config Config
	err := envconfig.Process("", &config)
	require.NoError(err)
	ctx := context.Background()
	s := newStore(ctx, t, config)
	defer s.Clear(ctx)

	assert := assert.New(t)

	err = s.InsertLinkedCloud(ctx, store.LinkedCloud{
		ID:           "testID",
		Name:         "testName",
		ClientID:     "testClientID",
		ClientSecret: "testClientSecret",
		Scopes:       []string{"testScopes"},
		Endpoint: store.Endpoint{
			AuthUrl:  "testAuthUrl",
			TokenUrl: "testTokenUrl",
		},
	})
	require.NoError(err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.RemoveLinkedCloud(ctx, tt.args.LinkedCloudId)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

type testLinkedCloudHandler struct {
	lcs []store.LinkedCloud
}

func (h *testLinkedCloudHandler) Handle(ctx context.Context, iter store.LinkedCloudIter) (err error) {
	var sub store.LinkedCloud
	for iter.Next(ctx, &sub) {
		h.lcs = append(h.lcs, sub)
	}
	return iter.Err()
}

func TestStore_LoadLinkedClouds(t *testing.T) {
	lcs := []store.LinkedCloud{store.LinkedCloud{
		ID:           "testID",
		Name:         "testName",
		ClientID:     "testClientID",
		ClientSecret: "testClientSecret",
		Scopes:       []string{"testScopes"},
		Audience:     "testAudience",
		Endpoint: store.Endpoint{
			AuthUrl:  "testAuthUrl",
			TokenUrl: "testTokenUrl",
		},
	},
		{
			ID:           "testID2",
			Name:         "testName",
			ClientID:     "testClientID",
			ClientSecret: "testClientSecret",
			Scopes:       []string{"testScopes"},
			Endpoint: store.Endpoint{
				AuthUrl:  "testAuthUrl",
				TokenUrl: "testTokenUrl",
			},
		},
	}

	type args struct {
		query store.Query
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []store.LinkedCloud
	}{
		{
			name: "all",
			args: args{
				query: store.Query{},
			},
			want: lcs,
		},
		{
			name: "id",
			args: args{
				query: store.Query{ID: lcs[0].ID},
			},
			want: []store.LinkedCloud{lcs[0]},
		},
		{
			name: "not found",
			args: args{
				query: store.Query{ID: "not found"},
			},
		},
	}

	require := require.New(t)
	var config Config
	err := envconfig.Process("", &config)
	require.NoError(err)
	ctx := context.Background()
	s := newStore(ctx, t, config)
	require.NoError(err)
	defer s.Clear(ctx)

	assert := assert.New(t)

	for _, l := range lcs {
		err = s.InsertLinkedCloud(ctx, l)
		require.NoError(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h testLinkedCloudHandler
			err := s.LoadLinkedClouds(ctx, tt.args.query, &h)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tt.want, h.lcs)
			}
		})
	}
}
