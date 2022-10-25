package mongodb_test

import (
	"context"
	"sync"
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/maintenance"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.opentelemetry.io/otel/trace"
)

type mockRecordHandler struct {
	lock  sync.Mutex
	tasks map[string]maintenance.Task
}

func newMockRecordHandler() *mockRecordHandler {
	return &mockRecordHandler{tasks: make(map[string]maintenance.Task)}
}

func (eh *mockRecordHandler) SetElement(aggregateID string, task maintenance.Task) {
	var aggregate maintenance.Task
	var ok bool

	eh.lock.Lock()
	defer eh.lock.Unlock()
	if aggregate, ok = eh.tasks[aggregateID]; !ok {
		eh.tasks[aggregateID] = maintenance.Task{GroupID: task.GroupID, AggregateID: task.AggregateID, Version: task.Version}
	}
	aggregate.GroupID = task.GroupID
	aggregate.AggregateID = task.AggregateID
	aggregate.Version = task.Version
}

func (eh *mockRecordHandler) Handle(ctx context.Context, iter maintenance.Iter) error {
	var task maintenance.Task

	for iter.Next(ctx, &task) {
		eh.SetElement(task.AggregateID, task)
	}
	return nil
}

func TestMaintenance(t *testing.T) {
	logger := log.NewLogger(log.MakeDefaultConfig())

	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	ctx := context.Background()

	store, err := mongodb.New(
		ctx,
		mongodb.Config{
			Embedded: pkgMongo.Config{
				URI: "mongodb://localhost:27017",
				TLS: config.MakeTLSClientConfig(),
			},
		},
		fileWatcher,
		logger,
		trace.NewNoopTracerProvider(),
		mongodb.WithMarshaler(bson.Marshal),
		mongodb.WithUnmarshaler(bson.Unmarshal),
	)
	assert.NoError(t, err)
	assert.NotNil(t, store)
	defer func() {
		t.Log("clearing db")
		errC := store.Clear(ctx)
		require.NoError(t, errC)
		_ = store.Close(ctx)
	}()

	const groupID = "groupID1"
	const aggregateID1 = "aggregateID1"
	tasksToSave := []maintenance.Task{
		{
			GroupID:     groupID,
			AggregateID: aggregateID1,
		},
		{
			GroupID:     groupID,
			AggregateID: aggregateID1,
			Version:     1,
		},
		{
			GroupID:     groupID,
			AggregateID: aggregateID1,
			Version:     2,
		},
		{
			GroupID:     groupID,
			AggregateID: aggregateID1,
			Version:     3,
		},
		{
			GroupID:     groupID,
			AggregateID: aggregateID1,
			Version:     4,
		},
	}

	t.Log("insert maintenance record without body")
	err = store.Insert(ctx, maintenance.Task{})
	require.Error(t, err)

	t.Log("insert maintenance record without GroupID")
	err = store.Insert(ctx, maintenance.Task{AggregateID: aggregateID1})
	require.Error(t, err)

	t.Log("insert maintenance record without AggregateID")
	err = store.Insert(ctx, maintenance.Task{GroupID: groupID})
	require.Error(t, err)

	t.Log("insert maintenance record")
	err = store.Insert(ctx, tasksToSave[1])
	require.NoError(t, err)

	t.Log("insert maintenance record with higher version")
	err = store.Insert(ctx, tasksToSave[4])
	require.NoError(t, err)

	t.Log("query maintenance records")
	eh1 := newMockRecordHandler()
	err = store.Query(ctx, 777, eh1)
	require.NoError(t, err)
	require.Equal(t, tasksToSave[4], eh1.tasks[aggregateID1])

	t.Log("insert maintenance record with lower version")
	err = store.Insert(ctx, tasksToSave[3])
	require.Error(t, err)

	t.Log("query maintenance records")
	eh2 := newMockRecordHandler()
	err = store.Query(ctx, 777, eh2)
	require.NoError(t, err)
	require.Equal(t, tasksToSave[4], eh2.tasks[aggregateID1])

	t.Log("remove maintenance record - incorrect version")
	err = store.Remove(ctx, tasksToSave[3])
	require.Error(t, err)

	t.Log("remove maintenance record")
	err = store.Remove(ctx, tasksToSave[4])
	require.NoError(t, err)

	t.Log("query maintenance records - empty collection")
	eh3 := newMockRecordHandler()
	err = store.Query(ctx, 777, eh3)
	require.NoError(t, err)
	require.Equal(t, 0, len(eh3.tasks))
}
