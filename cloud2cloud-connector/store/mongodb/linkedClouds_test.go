package mongodb_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/test"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2/oauth"
	"github.com/stretchr/testify/require"
)

func TestStoreUpdateLinkedCloud(t *testing.T) {
	type args struct {
		sub store.LinkedCloud
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid ID",
			args: args{
				sub: store.LinkedCloud{
					ID:   "testIDnotFound",
					Name: "testName",
					Endpoint: store.Endpoint{
						URL: "testTargetURL",
					},
					OAuth: oauth.Config{
						ClientID:     "testClientID",
						ClientSecret: "testClientSecret",
						Scopes:       []string{"testScopes"},
						AuthURL:      "testAuthUrl",
						TokenURL:     "testTokenUrl",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				sub: store.LinkedCloud{
					ID:   "testID",
					Name: "testNameUpdated",
					Endpoint: store.Endpoint{
						URL: "testTargetURL",
					},
					OAuth: oauth.Config{
						ClientID:     "testClientID",
						ClientSecret: "testClientSecret",
						Scopes:       []string{"testScopes"},
						AuthURL:      "testAuthUrl",
						TokenURL:     "testTokenUrl",
					},
				},
			},
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	err := s.InsertLinkedCloud(ctx, store.LinkedCloud{
		ID:   "testID",
		Name: "testName",
		Endpoint: store.Endpoint{
			URL: "testTargetURL",
		},
		OAuth: oauth.Config{
			ClientID:     "testClientID",
			ClientSecret: "testClientSecret",
			Scopes:       []string{"testScopes"},
			AuthURL:      "testAuthUrl",
			TokenURL:     "testTokenUrl",
		},
	})
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.UpdateLinkedCloud(ctx, tt.args.sub)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStoreRemoveLinkedCloud(t *testing.T) {
	type args struct {
		LinkedCloudID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid cloudId",
			args: args{
				LinkedCloudID: "notFound",
			},
			wantErr: true,
		},

		{
			name: "valid",
			args: args{
				LinkedCloudID: "testID",
			},
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	err := s.InsertLinkedCloud(ctx, store.LinkedCloud{
		ID:   "testID",
		Name: "testName",
		Endpoint: store.Endpoint{
			URL: "testTargetURL",
		},
		OAuth: oauth.Config{
			ClientID:     "testClientID",
			ClientSecret: "testClientSecret",
			Scopes:       []string{"testScopes"},
			AuthURL:      "testAuthUrl",
			TokenURL:     "testTokenUrl",
		},
	})
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.RemoveLinkedCloud(ctx, tt.args.LinkedCloudID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
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

func TestStoreLoadLinkedClouds(t *testing.T) {
	lcs := []store.LinkedCloud{
		{
			ID:   "testID",
			Name: "testName",
			Endpoint: store.Endpoint{
				URL: "testTargetURL",
			},
			OAuth: oauth.Config{
				ClientID:     "testClientID",
				ClientSecret: "testClientSecret",
				Scopes:       []string{"testScopes"},
				AuthURL:      "testAuthUrl",
				TokenURL:     "testTokenUrl",
			},
		},
		{
			ID:   "testID2",
			Name: "testName",
			Endpoint: store.Endpoint{
				URL: "testTargetURL",
			},
			OAuth: oauth.Config{
				ClientID:     "testClientID",
				ClientSecret: "testClientSecret",
				Scopes:       []string{"testScopes"},
				AuthURL:      "testAuthUrl",
				TokenURL:     "testTokenUrl",
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

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	for _, l := range lcs {
		err := s.InsertLinkedCloud(ctx, l)
		require.NoError(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h testLinkedCloudHandler
			err := s.LoadLinkedClouds(ctx, tt.args.query, &h)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, h.lcs)
		})
	}
}
