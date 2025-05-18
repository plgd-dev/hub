package cqldb_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/cqldb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addEventsForDeleteToDB(ctx context.Context, t *testing.T, store *cqldb.EventStore) int {
	const eventCount = 2000
	const deviceCount = 10
	const resourceCount = 100
	var resourceVersion [resourceCount]uint64
	var resourceTimestamp [resourceCount]int64
	var resourceEvents [resourceCount][]eventstore.Event
	for i := range eventCount {
		deviceIndex := i % deviceCount
		resourceIndex := i % resourceCount
		if i < resourceCount {
			resourceTimestamp[i] = int64((eventCount / resourceCount) * i)
		}

		resourceEvents[resourceIndex] = append(resourceEvents[resourceIndex], test.MockEvent{
			VersionI:     resourceVersion[resourceIndex],
			EventTypeI:   "testType",
			IsSnapshotI:  true,
			AggregateIDI: getResourceID(deviceIndex, resourceIndex),
			GroupIDI:     getDeviceID(deviceIndex),
			TimestampI:   1 + resourceTimestamp[resourceIndex],
		})

		resourceVersion[resourceIndex]++
		resourceTimestamp[resourceIndex]++
	}

	for _, v := range resourceEvents {
		saveStatus, err := store.Save(ctx, v...)
		require.NoError(t, err)
		require.Equal(t, eventstore.Ok, saveStatus)
	}

	return eventCount
}

func TestEventStore_Delete(t *testing.T) {
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
	assert.NotNil(t, store)
	defer func() {
		t.Log("clearing db")
		errC := store.Clear(ctx)
		require.NoError(t, errC)
		_ = store.Close(ctx)
	}()

	type args struct {
		query []eventstore.DeleteQuery
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Invalid query",
			args: args{
				query: []eventstore.DeleteQuery{},
			},
			wantErr: true,
		},
		{
			name: "Invalid groupID",
			args: args{
				query: []eventstore.DeleteQuery{{
					GroupID: uuid.Nil.String(),
				}},
			},
			wantErr: false,
		},
		{
			name: "Invalid and valid groupID",
			args: args{
				query: []eventstore.DeleteQuery{{
					GroupID: uuid.Nil.String(),
				}, {
					GroupID: getDeviceID(1),
				}},
			},
			wantErr: false,
		},
		{
			name: "Delete single device",
			args: args{
				query: []eventstore.DeleteQuery{{
					GroupID: getDeviceID(5),
				}},
			},
			wantErr: false,
		},
		{
			name: "Delete multiple devices",
			args: args{
				query: []eventstore.DeleteQuery{{
					GroupID: getDeviceID(2),
				}, {
					GroupID: getDeviceID(3),
				}, {
					GroupID: getDeviceID(5),
				}, {
					GroupID: getDeviceID(7),
				}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addEventsForDeleteToDB(ctx, t, store)
			defer func() {
				err = store.ClearTable(ctx)
				require.NoError(t, err)
			}()

			err := store.Delete(ctx, tt.args.query)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// get all events after deletion
			handler := test.NewMockEventHandler()
			err = store.GetEvents(ctx, []eventstore.GetEventsQuery{{}}, 0, handler)
			require.NoError(t, err)
			// no documents with deleted group id should remain
			for _, q := range tt.args.query {
				require.False(t, handler.ContainsGroupID(q.GroupID))
			}
		})
	}
}
