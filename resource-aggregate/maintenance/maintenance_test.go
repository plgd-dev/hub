package maintenance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/snappy"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/maintenance"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.opentelemetry.io/otel/trace"
)

type mockEvent struct {
	VersionI     uint64
	EventTypeI   string
	AggregateIDI string
	groupID      string
	timestamp    int64
	Data         []byte
	ETagI        []byte
}

func (e mockEvent) Version() uint64 {
	return e.VersionI
}

func (e mockEvent) EventType() string {
	return e.EventTypeI
}

func (e mockEvent) AggregateID() string {
	return e.AggregateIDI
}

func (e mockEvent) GroupID() string {
	return e.groupID
}

func (e mockEvent) IsSnapshot() bool {
	return false
}

func (e mockEvent) ETag() *eventstore.ETagData {
	if e.ETagI == nil {
		return nil
	}
	return &eventstore.ETagData{
		ETag:      e.ETagI,
		Timestamp: e.timestamp,
	}
}

func (e mockEvent) Timestamp() time.Time {
	return time.Unix(0, e.timestamp)
}

func (e mockEvent) Marshal() ([]byte, error) {
	src, err := bson.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal event: %w", err)
	}
	dst := make([]byte, 1024)
	return snappy.Encode(dst, src), nil
}

func (e mockEvent) ServiceID() (string, bool) {
	return "", false
}

func getEventsToSave(groupID, aggregateID string) []eventstore.Event {
	resId := commands.NewResourceID(groupID, aggregateID)
	const timestamp = 12345
	return []eventstore.Event{
		&events.ResourceStateSnapshotTaken{
			ResourceId: resId,
			EventMetadata: &events.EventMetadata{
				Version:   0,
				Timestamp: timestamp,
			},
		},
		mockEvent{
			groupID:      resId.GetDeviceId(),
			AggregateIDI: resId.ToUUID().String(),
			VersionI:     1,
			EventTypeI:   "test1",
			timestamp:    timestamp,
		},
		mockEvent{
			groupID:      resId.GetDeviceId(),
			AggregateIDI: resId.ToUUID().String(),
			VersionI:     2,
			EventTypeI:   "test2",
			timestamp:    timestamp,
			Data:         []byte("data of event 2"),
		},
		mockEvent{
			groupID:      resId.GetDeviceId(),
			AggregateIDI: resId.ToUUID().String(),
			VersionI:     3,
			EventTypeI:   "test3",
			timestamp:    timestamp,
			Data:         []byte("data of event 3"),
		},
		mockEvent{
			groupID:      resId.GetDeviceId(),
			AggregateIDI: resId.ToUUID().String(),
			VersionI:     4,
			EventTypeI:   "test4",
			timestamp:    timestamp,
			Data:         []byte("data of event 4"),
		},
	}
}

func getTaskToSave(groupID, aggregateID string, version uint64) maintenance.Task {
	resId := commands.NewResourceID(groupID, aggregateID)
	return maintenance.Task{
		GroupID:     resId.GetDeviceId(),
		AggregateID: resId.ToUUID().String(),
		Version:     version,
	}
}

func TestPerformMaintenance(t *testing.T) {
	logger := log.NewLogger(log.MakeDefaultConfig())
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	config := Config{
		NumAggregates: 77,
		BackupPath:    "/tmp/events.txt",
		Mongo: mongodb.Config{
			Embedded: pkgMongo.Config{
				URI:      "mongodb://localhost:27017",
				Database: "maintenance_test",
				TLS:      config.MakeTLSClientConfig(),
			},
		},
	}

	ctx := context.Background()
	store, err := mongodb.New(
		ctx,
		config.Mongo,
		fileWatcher,
		logger,
		trace.NewNoopTracerProvider(),
		mongodb.WithMarshaler(bson.Marshal),
		mongodb.WithUnmarshaler(unmarshalPlain),
	)
	require.NoError(t, err)
	require.NotNil(t, store)

	defer func() {
		t.Log("clearing db")
		errC := store.Clear(ctx)
		require.NoError(t, errC)
		_ = store.Close(ctx)
	}()

	const groupID = "device1"
	const aggregateID1 = "aggregateID1"
	t.Log("insert aggregateID = 1 events into the event store")
	saveStatus, err := store.Save(ctx, getEventsToSave(groupID, aggregateID1)...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	const aggregateID2 = "aggregateID2"
	t.Log("insert aggregateID = 2 events into the event store")
	saveStatus, err = store.Save(ctx, getEventsToSave(groupID, aggregateID2)...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	tasksToSave := []maintenance.Task{
		getTaskToSave(groupID, aggregateID1, 3),
		getTaskToSave(groupID, aggregateID1, 4),
		getTaskToSave(groupID, aggregateID2, 3),
	}

	t.Log("perform maintenance")
	err = store.Insert(ctx, tasksToSave[0])
	require.NoError(t, err)

	err = performMaintenanceWithEventStore(ctx, config, store)
	require.NoError(t, err)

	t.Log("perform maintenance again")
	err = store.Insert(ctx, tasksToSave[1])
	require.NoError(t, err)
	err = store.Insert(ctx, tasksToSave[2])
	require.NoError(t, err)

	err = performMaintenanceWithEventStore(ctx, config, store)
	require.NoError(t, err)
}
