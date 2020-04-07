package mongodb

import (
	"context"
	"testing"

	"github.com/go-ocf/cloud/openapi-connector/store"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_InsertLinkedAccount(t *testing.T) {
	type args struct {
		sub store.LinkedAccount
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				sub: store.LinkedAccount{
					ID:        "testID",
					TargetURL: "testTargetURL",
					TargetCloud: store.OAuth{
						LinkedCloudID: "testLinkedCloudID",
						AccessToken:   "testAccessToken",
						RefreshToken:  "testRefreshToken",
					},
					OriginCloud: store.OAuth{
						LinkedCloudID: "testLinkedCloudID",
						AccessToken:   "testAccessToken",
						RefreshToken:  "testRefreshToken",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.InsertLinkedAccount(ctx, tt.args.sub)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestStore_UpdateLinkedAccount(t *testing.T) {
	type args struct {
		sub store.LinkedAccount
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "not found",
			args: args{
				sub: store.LinkedAccount{
					ID:        "testID1",
					TargetURL: "testTargetURL",
					TargetCloud: store.OAuth{
						LinkedCloudID: "testLinkedCloudID",
						AccessToken:   "testAccessToken",
						RefreshToken:  "testRefreshToken",
					},
					OriginCloud: store.OAuth{
						LinkedCloudID: "testLinkedCloudID",
						AccessToken:   "testAccessToken",
						RefreshToken:  "testRefreshToken",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				sub: store.LinkedAccount{
					ID:        "testID",
					TargetURL: "testTargetURL",
					TargetCloud: store.OAuth{
						LinkedCloudID: "testLinkedCloudID",
						AccessToken:   "testAccessToken",
						RefreshToken:  "testRefreshToken",
					},
					OriginCloud: store.OAuth{
						LinkedCloudID: "testLinkedCloudID",
						AccessToken:   "testAccessToken",
						RefreshToken:  "testRefreshToken",
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

	err = s.InsertLinkedAccount(ctx, store.LinkedAccount{
		ID:        "testID",
		TargetURL: "testTargetURL",
		TargetCloud: store.OAuth{
			LinkedCloudID: "testLinkedCloudID",
			AccessToken:   "testAccessToken",
			RefreshToken:  "testRefreshToken",
		},
		OriginCloud: store.OAuth{
			LinkedCloudID: "testLinkedCloudID",
			AccessToken:   "testAccessToken",
			RefreshToken:  "testRefreshToken",
		},
	})
	require.NoError(err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.UpdateLinkedAccount(ctx, tt.args.sub)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestStore_RemoveLinkedAccount(t *testing.T) {
	type args struct {
		linkedAccountId string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "not found",
			args: args{
				linkedAccountId: "testNotFound",
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				linkedAccountId: "testID",
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

	err = s.InsertLinkedAccount(ctx, store.LinkedAccount{
		ID:        "testID",
		TargetURL: "testTargetURL",
		TargetCloud: store.OAuth{
			LinkedCloudID: "testLinkedCloudID",
			AccessToken:   "testAccessToken",
			RefreshToken:  "testRefreshToken",
		},
		OriginCloud: store.OAuth{
			LinkedCloudID: "testLinkedCloudID",
			AccessToken:   "testAccessToken",
			RefreshToken:  "testRefreshToken",
		},
	})
	require.NoError(err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.RemoveLinkedAccount(ctx, tt.args.linkedAccountId)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

type testLinkedAccountHandler struct {
	accs []store.LinkedAccount
}

func (h *testLinkedAccountHandler) Handle(ctx context.Context, iter store.LinkedAccountIter) (err error) {
	var sub store.LinkedAccount
	for iter.Next(ctx, &sub) {
		h.accs = append(h.accs, sub)
	}
	return iter.Err()
}

func TestStore_LoadLinkedAccounts(t *testing.T) {
	linkedAccounts := []store.LinkedAccount{
		store.LinkedAccount{
			ID:        "testID",
			TargetURL: "testTargetURL",
			TargetCloud: store.OAuth{
				LinkedCloudID: "testLinkedCloudID",
				AccessToken:   "testAccessToken",
				RefreshToken:  "testRefreshToken",
			},
			OriginCloud: store.OAuth{
				LinkedCloudID: "testLinkedCloudID",
				AccessToken:   "testAccessToken",
				RefreshToken:  "testRefreshToken",
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
		want    []store.LinkedAccount
	}{
		{
			name: "all",
			args: args{
				query: store.Query{},
			},
			want: linkedAccounts,
		},
		{
			name: "id",
			args: args{
				query: store.Query{ID: linkedAccounts[0].ID},
			},
			want: []store.LinkedAccount{linkedAccounts[0]},
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

	for _, a := range linkedAccounts {
		err = s.InsertLinkedAccount(ctx, a)
		require.NoError(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h testLinkedAccountHandler
			err := s.LoadLinkedAccounts(ctx, tt.args.query, &h)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tt.want, h.accs)
			}
		})
	}
}
