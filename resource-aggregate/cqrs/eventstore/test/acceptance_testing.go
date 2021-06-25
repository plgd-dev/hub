package test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/stretchr/testify/require"
)

type mockEvent struct {
	VersionI    uint64 `bson:"version"`
	EventTypeI  string `bson:"eventtype"`
	isSnapshot  bool   `bson:"issnapshot"`
	aggregateID string `bson:"aggregateid"`
	groupID     string `bson:"groupid"`
	Data        []byte `bson:"data"`
	timestamp   int64  `bson:"timestamp"`
}

func (e mockEvent) Version() uint64 {
	return e.VersionI
}

func (e mockEvent) EventType() string {
	return e.EventTypeI
}

func (e mockEvent) AggregateID() string {
	return e.aggregateID
}

func (e mockEvent) GroupID() string {
	return e.groupID
}

func (e mockEvent) IsSnapshot() bool {
	return e.isSnapshot
}

func (e mockEvent) Timestamp() time.Time {
	return time.Unix(0, e.timestamp)
}

type mockEventHandler struct {
	lock   sync.Mutex
	events map[string]map[string][]eventstore.Event
}

func NewMockEventHandler() *mockEventHandler {
	return &mockEventHandler{events: make(map[string]map[string][]eventstore.Event)}
}

func (eh *mockEventHandler) SetElement(groupId, aggrageId string, e mockEvent) {
	var device map[string][]eventstore.Event
	var ok bool

	eh.lock.Lock()
	defer eh.lock.Unlock()
	if device, ok = eh.events[groupId]; !ok {
		device = make(map[string][]eventstore.Event)
		eh.events[groupId] = device
	}
	device[aggrageId] = append(device[aggrageId], e)
}

func (eh *mockEventHandler) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return errors.New("cannot determine type of event")
		}
		var e mockEvent
		err := eu.Unmarshal(&e)
		if err != nil {
			return err
		}
		e.aggregateID = eu.AggregateID()
		e.groupID = eu.GroupID()
		e.isSnapshot = eu.IsSnapshot()
		e.timestamp = eu.Timestamp().UnixNano()
		eh.SetElement(eu.GroupID(), eu.AggregateID(), e)
	}
	return nil
}

func (eh *mockEventHandler) SnapshotEventType() string { return "snapshot" }

// AcceptanceTest is the acceptance test that all implementations of EventStore
// should pass. It should manually be called from a test case in each
// implementation:
//
//   func TestEventStore(t *testing.T) {
//       ctx := context.Background() // Or other when testing namespaces.
//       store := NewEventStore()
//       eventstore.AcceptanceTest(t, ctx, store)
//   }
//

func getEvents(fromVersion uint64, num uint64, firstEventSnapshot bool, groupID string, aggregateID string, timestamp int64) []eventstore.Event {
	e := []eventstore.Event{
		mockEvent{
			VersionI:    fromVersion,
			EventTypeI:  "test0",
			aggregateID: aggregateID,
			groupID:     groupID,
			isSnapshot:  firstEventSnapshot,
			timestamp:   timestamp,
		},
	}
	for i := uint64(1); i < num; i++ {
		e = append(e, mockEvent{
			VersionI:    fromVersion + i,
			EventTypeI:  "test0",
			aggregateID: aggregateID,
			groupID:     groupID,
			timestamp:   timestamp + int64(i),
		})
	}
	return e
}

func AcceptanceTest(t *testing.T, ctx context.Context, store eventstore.EventStore) {
	AggregateID1 := "aggregateID1"
	AggregateID2 := "aggregateID2"
	AggregateID3 := "aggregateID3"
	AggregateID4 := "aggregateID4"
	type Path struct {
		GroupID     string
		AggregateID string
	}

	aggregateID1Path := Path{
		AggregateID: AggregateID1,
		GroupID:     "deviceId",
	}
	aggregateID2Path := Path{
		AggregateID: AggregateID2,
		GroupID:     "deviceId",
	}
	aggregateID3Path := Path{
		AggregateID: AggregateID3,
		GroupID:     "deviceId1",
	}
	aggregateID4Path := Path{
		AggregateID: AggregateID4,
		GroupID:     "deviceId2",
	}

	timestamp := time.Date(2021, time.April, 1, 13, 37, 00, 0, time.UTC).UnixNano()

	t.Log("save no events")
	saveStatus, err := store.Save(ctx, nil)
	require.Error(t, err)
	require.Equal(t, eventstore.Fail, saveStatus)

	t.Log("save event, VersionI 0")
	saveStatus, err = store.Save(ctx, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[0])
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	t.Log("save event, VersionI 1")
	saveStatus, err = store.Save(ctx, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[1])
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	t.Log("try to save same event VersionI 1 twice")
	saveStatus, err = store.Save(ctx, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[1])
	require.NoError(t, err)
	require.Equal(t, eventstore.ConcurrencyException, saveStatus)

	t.Log("save event, VersionI 2")
	saveStatus, err = store.Save(ctx, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[2])
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	t.Log("save multiple events, VersionI 3, 4 and 5")
	saveStatus, err = store.Save(ctx, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[3:6]...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	t.Log("save event for another aggregate")
	saveStatus, err = store.Save(ctx, getEvents(0, 6, false, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp)[0:4]...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	t.Log("save events and then save snapshot with events")
	saveStatus, err = store.Save(ctx, getEvents(0, 3, false, aggregateID4Path.GroupID, aggregateID4Path.AggregateID, timestamp)...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	saveStatus, err = store.Save(ctx, getEvents(3, 4, true, aggregateID4Path.GroupID, aggregateID4Path.AggregateID, timestamp)...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	t.Log("load events from snapshot")
	saveEh := NewMockEventHandler()
	err = store.LoadFromSnapshot(ctx, []eventstore.SnapshotQuery{
		{
			GroupID:     aggregateID4Path.GroupID,
			AggregateID: aggregateID4Path.AggregateID,
		},
	}, saveEh)
	require.NoError(t, err)
	require.Equal(t, getEvents(3, 4, true, aggregateID4Path.GroupID, aggregateID4Path.AggregateID, timestamp), saveEh.events[aggregateID4Path.GroupID][aggregateID4Path.AggregateID])

	t.Log("test if need snapshot occurs from save")
	bigEv := getEvents(7, 1, false, aggregateID4Path.GroupID, aggregateID4Path.AggregateID, timestamp)[0].(mockEvent)
	bigEv.Data = make([]byte, 7*1024*1024)

	saveStatus, err = store.Save(ctx, bigEv)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	bigEv.VersionI++
	saveStatus, err = store.Save(ctx, bigEv)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	bigEv.VersionI++
	saveStatus, err = store.Save(ctx, bigEv)
	require.NoError(t, err)
	require.Equal(t, eventstore.SnapshotRequired, saveStatus)

	bigEv.isSnapshot = true
	saveStatus, err = store.Save(ctx, bigEv)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)
	exp := []eventstore.Event{bigEv}

	bigEv.VersionI++
	bigEv.isSnapshot = false
	saveStatus, err = store.Save(ctx, bigEv)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)
	exp = append(exp, bigEv)

	saveEh1 := NewMockEventHandler()
	err = store.LoadFromSnapshot(ctx, []eventstore.SnapshotQuery{
		{
			GroupID:     aggregateID4Path.GroupID,
			AggregateID: aggregateID4Path.AggregateID,
		},
	}, saveEh1)
	require.NoError(t, err)
	require.Equal(t, exp, saveEh1.events[aggregateID4Path.GroupID][aggregateID4Path.AggregateID])

	t.Log("load events for non-existing aggregate")
	eh1 := NewMockEventHandler()
	err = store.LoadFromSnapshot(ctx, []eventstore.SnapshotQuery{{GroupID: "notExist"}}, eh1)
	require.NoError(t, err)
	require.Equal(t, 0, len(eh1.events))

	t.Log("load events")
	eh2 := NewMockEventHandler()
	err = store.LoadFromSnapshot(ctx, []eventstore.SnapshotQuery{
		{
			GroupID:     aggregateID1Path.GroupID,
			AggregateID: aggregateID1Path.AggregateID,
		},
	}, eh2)
	require.NoError(t, err)
	require.Equal(t, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[:6], eh2.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])

	t.Log("load events from version")
	eh3 := NewMockEventHandler()
	err = store.LoadFromVersion(ctx, []eventstore.VersionQuery{
		{
			GroupID:     aggregateID1Path.GroupID,
			AggregateID: aggregateID1Path.AggregateID,
			Version:     getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[2].Version(),
		},
	}, eh3)
	require.NoError(t, err)
	require.Equal(t, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[2:6], eh3.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])

	t.Log("load multiple aggregates by all queries")
	eh4 := NewMockEventHandler()
	err = store.LoadFromVersion(ctx, []eventstore.VersionQuery{
		{
			GroupID:     aggregateID1Path.GroupID,
			AggregateID: aggregateID1Path.AggregateID,
		},
		{
			GroupID:     aggregateID2Path.GroupID,
			AggregateID: aggregateID2Path.AggregateID,
		},
	}, eh4)
	require.NoError(t, err)
	require.Equal(t, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[0:6], eh4.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])
	require.Equal(t, getEvents(0, 6, false, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp)[0:4], eh4.events[aggregateID2Path.GroupID][aggregateID2Path.AggregateID])

	t.Log("load multiple aggregates by groupId")
	eh5 := NewMockEventHandler()
	err = store.LoadFromSnapshot(ctx, []eventstore.SnapshotQuery{
		{
			GroupID: aggregateID1Path.GroupID,
		},
	}, eh5)
	require.NoError(t, err)
	require.Equal(t,
		getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[0:6],
		eh5.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])
	require.Equal(t,
		getEvents(0, 6, false, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp)[0:4],
		eh5.events[aggregateID2Path.GroupID][aggregateID2Path.AggregateID])

	t.Log("load multiple aggregates by all")
	eh6 := NewMockEventHandler()
	saveStatus, err = store.Save(ctx, getEvents(0, 6, false, aggregateID3Path.GroupID, aggregateID3Path.AggregateID, timestamp)[0])
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)
	err = store.LoadFromSnapshot(ctx, []eventstore.SnapshotQuery{{GroupID: aggregateID1Path.GroupID}, {GroupID: aggregateID2Path.GroupID}, {GroupID: aggregateID3Path.GroupID}}, eh6)
	require.NoError(t, err)
	require.Equal(t, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[0:6],
		eh6.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])
	require.Equal(t,
		getEvents(0, 6, false, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp)[0:4],
		eh6.events[aggregateID2Path.GroupID][aggregateID2Path.AggregateID])
	require.Equal(t, []eventstore.Event{
		getEvents(0, 6, false, aggregateID3Path.GroupID, aggregateID3Path.AggregateID, timestamp)[0],
	}, eh6.events[aggregateID3Path.GroupID][aggregateID3Path.AggregateID])

	t.Log("load events up to version")
	eh7 := NewMockEventHandler()
	err = store.LoadUpToVersion(ctx, []eventstore.VersionQuery{
		{
			GroupID:     aggregateID1Path.GroupID,
			AggregateID: aggregateID1Path.AggregateID,
			Version:     getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[5].Version(),
		},
	}, eh7)
	require.NoError(t, err)
	require.Equal(t, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[0:5], eh7.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])

	t.Log("load events up to version")
	eh8 := NewMockEventHandler()
	err = store.LoadUpToVersion(ctx, []eventstore.VersionQuery{
		{
			GroupID:     aggregateID1Path.GroupID,
			AggregateID: aggregateID1Path.AggregateID,
			Version:     getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[0].Version(),
		},
	}, eh8)
	require.NoError(t, err)
	require.Equal(t, 0, len(eh8.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID]))

	t.Log("load events up to version without version specified")
	eh9 := NewMockEventHandler()
	err = store.LoadUpToVersion(ctx, []eventstore.VersionQuery{
		{
			GroupID:     aggregateID1Path.GroupID,
			AggregateID: aggregateID1Path.AggregateID,
		},
	}, eh9)
	require.NoError(t, err)
	require.Equal(t, 0, len(eh9.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID]))

	t.Log("test projection all")
	model := NewMockEventHandler()
	p := eventstore.NewProjection(store, func(context.Context, string, string) (eventstore.Model, error) { return model, nil }, nil)

	err = p.Project(ctx, []eventstore.SnapshotQuery{{GroupID: aggregateID1Path.GroupID}, {GroupID: aggregateID2Path.GroupID}, {GroupID: aggregateID3Path.GroupID}})
	require.NoError(t, err)
	require.Equal(t, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[0:6], model.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])
	require.Equal(t, getEvents(0, 6, false, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp)[0:4], model.events[aggregateID2Path.GroupID][aggregateID2Path.AggregateID])
	require.Equal(t, []eventstore.Event{
		getEvents(0, 6, false, aggregateID3Path.GroupID, aggregateID3Path.AggregateID, timestamp)[0],
	}, model.events[aggregateID3Path.GroupID][aggregateID3Path.AggregateID])

	t.Log("test projection group")
	model1 := NewMockEventHandler()
	p = eventstore.NewProjection(store, func(context.Context, string, string) (eventstore.Model, error) { return model1, nil }, nil)

	err = p.Project(ctx, []eventstore.SnapshotQuery{{GroupID: aggregateID1Path.GroupID}})
	require.NoError(t, err)
	require.Equal(t, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[0:6], model1.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])
	require.Equal(t, getEvents(0, 6, false, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp)[0:4], model1.events[aggregateID2Path.GroupID][aggregateID2Path.AggregateID])

	t.Log("test projection aggregate")
	model2 := NewMockEventHandler()
	p = eventstore.NewProjection(store, func(context.Context, string, string) (eventstore.Model, error) { return model2, nil }, nil)

	err = p.Project(ctx, []eventstore.SnapshotQuery{
		{
			GroupID:     aggregateID2Path.GroupID,
			AggregateID: aggregateID2Path.AggregateID,
		},
	})
	require.NoError(t, err)
	require.Equal(t, getEvents(0, 6, false, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp)[0:4], model2.events[aggregateID2Path.GroupID][aggregateID2Path.AggregateID])

	t.Log("remove events up to version")
	versionToRemove := 3
	err = store.RemoveUpToVersion(ctx, []eventstore.VersionQuery{
		{
			GroupID:     aggregateID1Path.GroupID,
			AggregateID: aggregateID1Path.AggregateID,
			Version:     getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[versionToRemove].Version(),
		},
	})
	require.NoError(t, err)

	eh10 := NewMockEventHandler()
	err = store.LoadFromVersion(ctx, []eventstore.VersionQuery{
		{
			GroupID:     aggregateID1Path.GroupID,
			AggregateID: aggregateID1Path.AggregateID,
		},
	}, eh10)
	require.NoError(t, err)
	require.Equal(t, getEvents(0, 6, false, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[versionToRemove:6], eh10.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])
}
