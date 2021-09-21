package mongodb_test

// import (
// 	"context"
// 	"testing"

// 	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
// 	"github.com/plgd-dev/cloud/cloud2cloud-connector/test"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestStore_InsertLinkedAccount(t *testing.T) {
// 	type args struct {
// 		sub store.LinkedAccount
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		wantErr bool
// 	}{
// 		{
// 			name: "valid",
// 			args: args{
// 				sub: store.LinkedAccount{
// 					ID:            "testID",
// 					LinkedCloudID: "testLinkedCloudID",
// 					TargetCloud: store.Token{
// 						AccessToken:  "testAccessToken",
// 						RefreshToken: "testRefreshToken",
// 					},
// 					UserID: "userID",
// 				},
// 			},
// 		},
// 	}

// 	s, cleanUpStore := test.NewMongoStore(t)
// 	defer cleanUpStore()

// 	ctx := context.Background()
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			err := s.InsertLinkedAccount(ctx, tt.args.sub)
// 			if tt.wantErr {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 			}
// 		})
// 	}
// }

// func TestStore_UpdateLinkedAccount(t *testing.T) {
// 	type args struct {
// 		sub store.LinkedAccount
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		wantErr bool
// 	}{
// 		{
// 			name: "not found",
// 			args: args{
// 				sub: store.LinkedAccount{
// 					ID:            "testID1",
// 					LinkedCloudID: "testLinkedCloudID",
// 					TargetCloud: store.Token{
// 						AccessToken:  "testAccessToken",
// 						RefreshToken: "testRefreshToken",
// 					},
// 					UserID: "userID",
// 				},
// 			},
// 			wantErr: true,
// 		},
// 		{
// 			name: "valid",
// 			args: args{
// 				sub: store.LinkedAccount{
// 					ID: "testID",

// 					LinkedCloudID: "testLinkedCloudID",
// 					TargetCloud: store.Token{
// 						AccessToken:  "testAccessToken",
// 						RefreshToken: "testRefreshToken",
// 					},
// 					UserID: "userID",
// 				},
// 			},
// 		},
// 	}

// 	s, cleanUpStore := test.NewMongoStore(t)
// 	defer cleanUpStore()

// 	ctx := context.Background()
// 	err := s.InsertLinkedAccount(ctx, store.LinkedAccount{
// 		ID: "testID",

// 		LinkedCloudID: "testLinkedCloudID",
// 		TargetCloud: store.Token{
// 			AccessToken:  "testAccessToken",
// 			RefreshToken: "testRefreshToken",
// 		},
// 		UserID: "userID",
// 	})
// 	require.NoError(t, err)

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			err := s.UpdateLinkedAccount(ctx, tt.args.sub)
// 			if tt.wantErr {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 			}
// 		})
// 	}
// }

// func TestStore_RemoveLinkedAccount(t *testing.T) {
// 	type args struct {
// 		linkedAccountId string
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		wantErr bool
// 	}{
// 		{
// 			name: "not found",
// 			args: args{
// 				linkedAccountId: "testNotFound",
// 			},
// 			wantErr: true,
// 		},
// 		{
// 			name: "valid",
// 			args: args{
// 				linkedAccountId: "testID",
// 			},
// 		},
// 	}

// 	s, cleanUpStore := test.NewMongoStore(t)
// 	defer cleanUpStore()

// 	ctx := context.Background()
// 	err := s.InsertLinkedAccount(ctx, store.LinkedAccount{
// 		ID: "testID",

// 		LinkedCloudID: "testLinkedCloudID",
// 		TargetCloud: store.Token{
// 			AccessToken:  "testAccessToken",
// 			RefreshToken: "testRefreshToken",
// 		},
// 		UserID: "userID",
// 	})
// 	require.NoError(t, err)

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			err := s.RemoveLinkedAccount(ctx, tt.args.linkedAccountId)
// 			if tt.wantErr {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 			}
// 		})
// 	}
// }

// type testLinkedAccountHandler struct {
// 	accs []store.LinkedAccount
// }

// func (h *testLinkedAccountHandler) Handle(ctx context.Context, iter store.LinkedAccountIter) (err error) {
// 	for {
// 		var sub store.LinkedAccount
// 		if !iter.Next(ctx, &sub) {
// 			break
// 		}
// 		h.accs = append(h.accs, sub)
// 	}
// 	return iter.Err()
// }

// func TestStore_LoadLinkedAccounts(t *testing.T) {
// 	linkedAccounts := []store.LinkedAccount{
// 		{
// 			ID: "testID",

// 			LinkedCloudID: "testLinkedCloudID",
// 			TargetCloud: store.Token{
// 				AccessToken:  "testAccessToken",
// 				RefreshToken: "testRefreshToken",
// 			},
// 			UserID: "userID",
// 		},
// 	}

// 	type args struct {
// 		query store.Query
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		wantErr bool
// 		want    []store.LinkedAccount
// 	}{
// 		{
// 			name: "all",
// 			args: args{
// 				query: store.Query{},
// 			},
// 			want: linkedAccounts,
// 		},
// 		{
// 			name: "id",
// 			args: args{
// 				query: store.Query{ID: linkedAccounts[0].ID},
// 			},
// 			want: []store.LinkedAccount{linkedAccounts[0]},
// 		},
// 		{
// 			name: "not found",
// 			args: args{
// 				query: store.Query{ID: "not found"},
// 			},
// 		},
// 	}

// 	s, cleanUpStore := test.NewMongoStore(t)
// 	defer cleanUpStore()

// 	ctx := context.Background()
// 	for _, a := range linkedAccounts {
// 		err := s.InsertLinkedAccount(ctx, a)
// 		require.NoError(t, err)
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			var h testLinkedAccountHandler
// 			err := s.LoadLinkedAccounts(ctx, tt.args.query, &h)
// 			if tt.wantErr {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 				assert.Equal(t, tt.want, h.accs)
// 			}
// 		})
// 	}
// }
