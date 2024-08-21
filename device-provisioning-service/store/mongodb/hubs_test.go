package mongodb_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/require"
)

func TestStoreCreateHub(t *testing.T) {
	hubID := "id"
	owner := "owner"
	type args struct {
		owner string
		hub   *store.Hub
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid ID",
			args: args{
				owner: owner,
				hub:   &store.Hub{},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				owner: owner,
				hub:   test.NewHub(hubID, owner),
			},
		},
		{
			name: "duplicity",
			args: args{
				owner: owner,
				hub: &store.Hub{
					Id: hubID,
				},
			},
			wantErr: true,
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.CreateHub(ctx, tt.args.owner, tt.args.hub)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStoreUpdateHub(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()
	const owner = "owner"

	ctx := context.Background()
	hub := test.NewHub("id", owner)
	err := s.CreateHub(ctx, owner, hub)
	require.NoError(t, err)

	hub.Gateways = []string{"abc"}

	type args struct {
		owner string
		hub   *store.Hub
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid ID",
			args: args{
				owner: owner,
				hub:   &store.Hub{},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				owner: owner,
				hub:   hub,
			},
		},
		{
			name: "duplicity",
			args: args{
				owner: owner,
				hub:   hub,
			},
		},
		{
			name: "not exist",
			args: args{
				owner: owner,
				hub:   &store.Hub{Id: "notExist"},
			},
			wantErr: true,
		},
		{
			name: "another owner",
			args: args{
				owner: anotherOwner,
				hub:   hub,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.UpdateHub(ctx, tt.args.owner, tt.args.hub)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

type testHubHandler struct {
	lcs pb.Hubs
}

func (h *testHubHandler) Handle(ctx context.Context, iter store.HubIter) (err error) {
	for {
		var hub store.Hub
		if !iter.Next(ctx, &hub) {
			break
		}
		h.lcs = append(h.lcs, &hub)
	}
	return iter.Err()
}

func getHubs(ctx context.Context, s *mongodb.Store, owner string, query *pb.GetHubsRequest) (pb.Hubs, error) {
	var h testHubHandler
	err := s.LoadHubs(ctx, owner, query, h.Handle)
	if err != nil {
		return nil, err
	}
	return h.lcs, nil
}

func TestStoreDeleteHub(t *testing.T) {
	const owner = "owner"
	hubIDs := []string{"0", "1", "2"}
	owners := []string{owner, owner, anotherOwner}

	type args struct {
		owner string
		query *store.HubsQuery
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		count   int64
	}{
		{
			name: "invalid cloudId",
			args: args{
				owner: owner,
				query: &store.HubsQuery{
					IdFilter: []string{"notFound"},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				owner: owner,
				query: &store.HubsQuery{
					IdFilter: []string{hubIDs[0]},
				},
			},
			count: 1,
		},
		{
			name: "valid multiple",
			args: args{
				owner: owner,
				query: &store.HubsQuery{
					IdFilter: []string{hubIDs[1], hubIDs[2]},
				},
			},
			count: 1,
		},
		{
			name: "another owner",
			args: args{
				owner: "owner",
				query: &store.HubsQuery{
					IdFilter: []string{hubIDs[2]},
				},
			},
			wantErr: true,
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	for idx, id := range hubIDs {
		err := s.CreateHub(ctx, owners[idx], test.NewHub(id, owners[idx]))
		require.NoError(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.DeleteHubs(ctx, tt.args.owner, tt.args.query)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.count, got)

			// verify the absence of deleted hubs
			hubs, err := getHubs(ctx, s, tt.args.owner, tt.args.query)
			require.NoError(t, err)
			require.Empty(t, hubs)
		})
	}
}

func TestStoreLoadHubs(t *testing.T) {
	const owner = "owner"
	hubs := pb.Hubs{
		test.NewHub("id0", owner),
		test.NewHub("id1", owner),
		test.NewHub("id2", anotherOwner),
	}

	type args struct {
		owner string
		query *store.HubsQuery
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    pb.Hubs
	}{
		{
			name: "all",
			args: args{
				owner: owner,
				query: nil,
			},
			want: hubs[:2],
		},
		{
			name: "id",
			args: args{
				owner: owner,
				query: &store.HubsQuery{IdFilter: []string{hubs[1].GetId()}},
			},
			want: []*store.Hub{hubs[1]},
		},
		{
			name: "hubId",
			args: args{
				owner: owner,
				query: &store.HubsQuery{HubIdFilter: []string{hubs[1].GetHubId()}},
			},
			want: []*store.Hub{hubs[1]},
		},
		{
			name: "multiple queries",
			args: args{
				owner: owner,
				query: &store.HubsQuery{
					IdFilter:    []string{hubs[0].GetId(), hubs[2].GetId()},
					HubIdFilter: []string{hubs[1].GetHubId()},
				},
			},
			want: []*store.Hub{hubs[0], hubs[1]},
		},
		{
			name: "not found",
			args: args{
				owner: owner,
				query: &store.HubsQuery{IdFilter: []string{"not found"}},
			},
		},
		{
			name: "hubId - another owner",
			args: args{
				owner: anotherOwner,
				query: &store.HubsQuery{HubIdFilter: []string{hubs[1].GetHubId()}},
			},
		},
	}

	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx := context.Background()
	for _, l := range hubs {
		err := s.CreateHub(ctx, l.GetOwner(), l)
		require.NoError(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hubs, err := getHubs(ctx, s, tt.args.owner, tt.args.query)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, hubs, len(tt.want))
			hubs.Sort()
			want := make(pb.Hubs, len(tt.want))
			copy(want, tt.want)
			want.Sort()

			for i := range hubs {
				hubTest.CheckProtobufs(t, want[i], hubs[i], hubTest.RequireToCheckFunc(require.Equal))
			}
		})
	}
}
