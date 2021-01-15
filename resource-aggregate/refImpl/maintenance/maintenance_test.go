package maintenance

import (
	"fmt"

	"github.com/golang/snappy"
	"go.mongodb.org/mongo-driver/bson"
)

type mockEvent struct {
	VersionI   uint64 `bson:"version"`
	EventTypeI string `bson:"eventtype"`
	Data       []byte `bson:"data"`
}

func (e mockEvent) Version() uint64 {
	return e.VersionI
}

func (e mockEvent) EventType() string {
	return e.EventTypeI
}

func (e mockEvent) Marshal() ([]byte, error) {
	src, err := bson.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal event: %w", err)
	}
	dst := make([]byte, 1024)
	return snappy.Encode(dst, src), nil
}

// TODO: adapt Maintenance to use
/*
func TestPerformMaintenance(t *testing.T) {
	ctx := context.Background()

	config := Config{
		NumAggregates: 77,
		BackupPath:    "/tmp/events.txt",
		Mongo: cqrs.Config{
			URI:             "mongodb://localhost:27017",
			DatabaseName:    "maintenance_test",
			BatchSize:       16,
			MaxPoolSize:     16,
			MaxConnIdleTime: 240000000000,
		}}

	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)

	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)

	tlsConfig := dialCertManager.GetClientTLSConfig()

	store, err := cqrs.NewEventStore(config.Mongo, nil, mongodb.WithMarshaler(bson.Marshal), mongodb.WithUnmarshaler(unmarshalPlain), mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	require.NotNil(t, store)

	defer store.Close(ctx)
	defer func() {
		t.Log("clearing db")
		err := store.Clear(ctx)
		require.NoError(t, err)
	}()

	aggregateID1 := "aggregateID1"
	aggregateID2 := "aggregateID2"
	eventsToSave := []event.Event{
		&events.ResourceStateSnapshotTaken{
			ResourceStateSnapshotTaken: pb.ResourceStateSnapshotTaken{
				EventMetadata: &kitCqrsPb.EventMetadata{Version: 0},
			},
		},
		mockEvent{
			VersionI:   1,
			EventTypeI: "test1",
		},
		mockEvent{
			VersionI:   2,
			EventTypeI: "test2",
			Data:       []byte("data of event 2"),
		},
		mockEvent{
			VersionI:   3,
			EventTypeI: "test3",
			Data:       []byte("data of event 3"),
		},
		mockEvent{
			VersionI:   4,
			EventTypeI: "test4",
			Data:       []byte("data of event 4"),
		},
	}

	t.Log("insert aggregateID = 1 events into the event store")
	conExcep, err := store.Save(ctx, "default-group", aggregateID1, eventsToSave)
	require.NoError(t, err)
	require.False(t, conExcep)

	t.Log("insert aggregateID = 2 events into the event store")
	conExcep, err = store.Save(ctx, "default-group", aggregateID2, eventsToSave)
	require.NoError(t, err)
	require.False(t, conExcep)

	tasksToSave := []maintenance.Task{
		{
			AggregateID: aggregateID1,
			Version:     3,
		},
		{
			AggregateID: aggregateID1,
			Version:     4,
		},
		{
			AggregateID: aggregateID2,
			Version:     3,
		},
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
*/
