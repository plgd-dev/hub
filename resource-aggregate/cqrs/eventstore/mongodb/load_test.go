package mongodb_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/stretchr/testify/require"
)

func TestLoadFromSnapshot(t *testing.T) {
	type args struct {
		queries []eventstore.SnapshotQuery
	}
	type device struct {
		ID           string
		NumResources int
	}
	tests := []struct {
		name    string
		args    args
		want    []device
		wantErr bool
	}{
		{
			name: "All group events",
			args: args{
				queries: []eventstore.SnapshotQuery{
					{
						GroupID: getDeviceID(0),
					},
				},
			},
			want: []device{
				{
					ID:           getDeviceID(0),
					NumResources: 20,
				},
			},
		},
		{
			name: "Group events with aggregateID",
			args: args{
				queries: []eventstore.SnapshotQuery{
					{
						GroupID:     getDeviceID(0),
						AggregateID: getAggregateID(0),
					},
				},
			},
			want: []device{
				{
					ID:           getDeviceID(0),
					NumResources: 1,
				},
			},
		},
		{
			name: "Group events with type filter",
			args: args{
				queries: []eventstore.SnapshotQuery{
					{
						GroupID: getDeviceID(0),
						Types:   getTypes(0),
					},
					{
						GroupID: getDeviceID(1),
						Types:   getTypes(3)[0:1],
					},
				},
			},
			want: []device{
				{
					ID:           getDeviceID(0),
					NumResources: 7,
				},
				{
					ID:           getDeviceID(1),
					NumResources: 6,
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

	ctx := context.Background()
	store, err := NewTestEventStore(ctx, fileWatcher, logger)
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() {
		t.Log("clearing db")
		err = store.Clear(ctx)
		require.NoError(t, err)
		err := store.Close(ctx)
		require.NoError(t, err)
	}()

	_ = addEventsForGetEventsToDB(ctx, t, store)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewMockEventHandler()
			err := store.LoadFromSnapshot(ctx, tt.args.queries, h)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			events := h.PopEvents()
			require.Len(t, events, len(tt.want))
			for _, want := range tt.want {
				ags, ok := events[want.ID]
				require.True(t, ok)
				require.Len(t, ags, want.NumResources)
			}
		})
	}
}
