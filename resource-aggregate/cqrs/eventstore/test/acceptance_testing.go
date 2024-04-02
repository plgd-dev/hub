package test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/stretchr/testify/require"
)

// AcceptanceTest is the acceptance test that all implementations of EventStore
// should pass. It should manually be called from a test case in each
// implementation:
//
//   func TestEventStore(t *testing.T) {
//       ctx := context.Background() // Or other when testing namespaces.
//       store := NewEventStore()
//       test.AcceptanceTest(ctx, t,store)
//   }
//

func getEvents(fromVersion uint64, num uint64, groupID string, aggregateID string, timestamp int64) []eventstore.Event {
	e := []eventstore.Event{
		MockEvent{
			VersionI:     fromVersion,
			EventTypeI:   "test0",
			AggregateIDI: aggregateID,
			GroupIDI:     groupID,
			IsSnapshotI:  true,
			TimestampI:   timestamp,
		},
	}
	for i := uint64(1); i < num; i++ {
		e = append(e, MockEvent{
			VersionI:     fromVersion + i,
			EventTypeI:   "test0",
			AggregateIDI: aggregateID,
			GroupIDI:     groupID,
			TimestampI:   timestamp + int64(i),
			IsSnapshotI:  true,
		})
	}
	return e
}

type eventsFilter func(eventstore.Event) bool

func filterEvents(events []eventstore.Event, filter eventsFilter) []eventstore.Event {
	newEvents := make([]eventstore.Event, 0, len(events))
	for _, v := range events {
		if filter(v) {
			newEvents = append(newEvents, v)
		}
	}
	return newEvents
}

const (
	aggregateID1 = "a0000000-0000-0000-0000-000000000001"
	aggregateID2 = "a0000000-0000-0000-0000-000000000002"
	aggregateID3 = "a0000000-0000-0000-0000-000000000003"
	aggregateID4 = "a0000000-0000-0000-0000-000000000004"
)

var (
	groupID1 = "00000000-0000-0000-0000-000000000001"
	groupID2 = "00000000-0000-0000-0000-000000000002"
	groupID3 = "00000000-0000-0000-0000-000000000003"
)

func GetEventsTest(ctx context.Context, t *testing.T, store eventstore.EventStore) {
	t.Log("testing GetEvents")

	const timestamp1 = int64(0)
	const timestamp2 = int64(20)
	const timestamp3 = int64(40)
	const timestamp4 = int64(60)

	t.Log("insert events")
	groupID1Events := getEvents(0, 5, groupID1, aggregateID1, timestamp1)
	saveStatus, err := store.Save(ctx, groupID1Events...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)
	groupID1Events = groupID1Events[len(groupID1Events)-1:]

	groupID2AggID2Events := getEvents(0, 5, groupID2, aggregateID2, timestamp2)
	saveStatus, err = store.Save(ctx, groupID2AggID2Events...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)
	groupID2AggID3Events := getEvents(0, 5, groupID2, aggregateID3, timestamp3)
	saveStatus, err = store.Save(ctx, groupID2AggID3Events...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	groupID3Events := getEvents(0, 5, groupID3, aggregateID4, timestamp4)
	saveStatus, err = store.Save(ctx, groupID3Events...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	groupID3Events = groupID3Events[len(groupID3Events)-1:]
	groupID2Events := groupID2AggID2Events[len(groupID2AggID2Events)-1:]

	groupID2Events = append(groupID2Events, groupID2AggID3Events[len(groupID2AggID3Events)-1:]...)
	allEvents := groupID1Events
	allEvents = append(allEvents, groupID2Events...)
	allEvents = append(allEvents, groupID3Events...)

	t.Log("get all events")
	saveEh := NewMockEventHandler()
	err = store.GetEvents(ctx, []eventstore.GetEventsQuery{{}}, 0, saveEh)
	require.NoError(t, err)
	require.True(t, saveEh.Equals(allEvents))

	t.Logf("get groupid %v and %v events", groupID1, groupID2)
	saveEh = NewMockEventHandler()
	err = store.GetEvents(ctx, []eventstore.GetEventsQuery{{GroupID: groupID1}, {GroupID: groupID2}}, 0, saveEh)
	require.NoError(t, err)
	events := groupID1Events
	events = append(events, groupID2Events...)
	require.True(t, saveEh.Equals(events))

	t.Logf("get aggregateid %v events", aggregateID2)
	saveEh = NewMockEventHandler()
	err = store.GetEvents(ctx, []eventstore.GetEventsQuery{{AggregateID: aggregateID2}}, 0, saveEh)
	require.NoError(t, err)
	require.True(t, saveEh.Equals(groupID2AggID2Events[len(groupID2AggID2Events)-1:]))

	t.Logf("get groupid %v and aggregateid %v events", groupID1, aggregateID4)
	saveEh = NewMockEventHandler()
	err = store.GetEvents(ctx, []eventstore.GetEventsQuery{{GroupID: groupID1}, {GroupID: groupID3, AggregateID: aggregateID4}}, 0, saveEh)
	require.NoError(t, err)
	events = groupID1Events
	events = append(events, groupID3Events...)
	require.True(t, saveEh.Equals(events))

	timestamp := timestamp4 - 1
	t.Logf("get events with timestamp > %v", timestamp)
	saveEh = NewMockEventHandler()
	err = store.GetEvents(ctx, []eventstore.GetEventsQuery{{}}, timestamp, saveEh)
	require.NoError(t, err)
	require.True(t, saveEh.Equals(groupID3Events))

	timestamp = timestamp3 + 2
	t.Logf("get groupid (%v, %v) events with timestamp > %v", groupID2, groupID3, timestamp)
	saveEh = NewMockEventHandler()
	err = store.GetEvents(ctx, []eventstore.GetEventsQuery{{GroupID: groupID2}, {GroupID: groupID3}}, timestamp, saveEh)
	require.NoError(t, err)
	events = filterEvents(append(groupID2Events, groupID3Events...), func(e eventstore.Event) bool {
		return e.Timestamp().UnixNano() > timestamp
	})
	require.True(t, saveEh.Equals(events))

	timestamp = timestamp2 - 1
	t.Logf("get aggregateid (%v, %v) events with timestamp > %v", aggregateID3, aggregateID4, timestamp)
	saveEh = NewMockEventHandler()
	err = store.GetEvents(ctx, []eventstore.GetEventsQuery{{AggregateID: aggregateID3}, {AggregateID: aggregateID4}}, timestamp, saveEh)
	require.NoError(t, err)
	events = groupID2AggID3Events[len(groupID2AggID3Events)-1:]
	events = append(events, groupID3Events...)
	require.True(t, saveEh.Equals(events))
}

func emptySaveFailTest(ctx context.Context, t *testing.T, store eventstore.EventStore) {
	t.Log("try save no events")
	saveStatus, err := store.Save(ctx, nil)
	require.Error(t, err)
	require.Equal(t, eventstore.Fail, saveStatus)
}

func invalidTimpestampFailTest(ctx context.Context, t *testing.T, store eventstore.EventStore) {
	t.Log("try save descreasing timestamp")
	timestamp := time.Date(2021, time.April, 1, 13, 37, 0o0, 0, time.UTC).UnixNano()
	events := getEvents(0, 2, groupID1, aggregateID1, timestamp)
	mockEvent := events[1].(MockEvent)
	mockEvent.TimestampI = timestamp - 1
	events[1] = mockEvent
	saveStatus, err := store.Save(ctx, events...)
	require.Error(t, err)
	require.Equal(t, eventstore.Fail, saveStatus)
}

func AcceptanceTest(ctx context.Context, t *testing.T, store eventstore.EventStore) {
	type Path struct {
		GroupID     string
		AggregateID string
	}

	aggregateID1Path := Path{
		AggregateID: aggregateID1,
		GroupID:     groupID1,
	}
	aggregateID2Path := Path{
		AggregateID: aggregateID2,
		GroupID:     groupID1,
	}
	aggregateID3Path := Path{
		AggregateID: aggregateID3,
		GroupID:     groupID2,
	}
	aggregateID4Path := Path{
		AggregateID: aggregateID4,
		GroupID:     groupID3,
	}

	timestamp := time.Date(2021, time.April, 1, 13, 37, 0o0, 0, time.UTC).UnixNano()

	emptySaveFailTest(ctx, t, store)
	invalidTimpestampFailTest(ctx, t, store)

	t.Log("save event, VersionI 0")
	saveStatus, err := store.Save(ctx, getEvents(0, 6, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[0])
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	t.Log("save event, VersionI 0")
	saveStatus, err = store.Save(ctx, getEvents(0, 6, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[0])
	require.NoError(t, err)
	require.Equal(t, eventstore.ConcurrencyException, saveStatus)

	t.Log("save event, VersionI 1")
	saveStatus, err = store.Save(ctx, getEvents(0, 6, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[1])
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	t.Log("try to save same event VersionI 1 twice")
	saveStatus, err = store.Save(ctx, getEvents(0, 6, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[1])
	require.NoError(t, err)
	require.Equal(t, eventstore.ConcurrencyException, saveStatus)

	t.Log("save event, VersionI 2")
	saveStatus, err = store.Save(ctx, getEvents(0, 6, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[2])
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	t.Log("save multiple events, VersionI 3, 4 and 5")
	saveStatus, err = store.Save(ctx, getEvents(0, 6, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[3:6]...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	t.Log("save event for another aggregate")
	saveStatus, err = store.Save(ctx, getEvents(0, 6, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp)[0:4]...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	t.Log("save events and then save snapshot with events")
	saveStatus, err = store.Save(ctx, getEvents(0, 3, aggregateID4Path.GroupID, aggregateID4Path.AggregateID, timestamp)...)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	saveStatus, err = store.Save(ctx, getEvents(3, 4, aggregateID4Path.GroupID, aggregateID4Path.AggregateID, timestamp)...)
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
	require.Equal(t, getEvents(6, 1, aggregateID4Path.GroupID, aggregateID4Path.AggregateID, timestamp+3), saveEh.events[aggregateID4Path.GroupID][aggregateID4Path.AggregateID])

	t.Log("test with big snapshot")
	bigEv := getEvents(7, 1, aggregateID4Path.GroupID, aggregateID4Path.AggregateID, timestamp)[0].(MockEvent)
	bigEv.DataI = make([]byte, 7*1024*1024)

	saveStatus, err = store.Save(ctx, bigEv)
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)

	exp := []eventstore.Event{bigEv}
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
	err = store.LoadFromSnapshot(ctx, []eventstore.SnapshotQuery{{GroupID: uuid.Nil.String()}}, eh1)
	require.NoError(t, err)
	require.Empty(t, eh1.events)

	t.Log("load events")
	eh2 := NewMockEventHandler()
	err = store.LoadFromSnapshot(ctx, []eventstore.SnapshotQuery{
		{
			GroupID:     aggregateID1Path.GroupID,
			AggregateID: aggregateID1Path.AggregateID,
		},
	}, eh2)
	require.NoError(t, err)
	require.Equal(t, getEvents(5, 1, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp+5), eh2.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])

	t.Log("load events from version")
	eh3 := NewMockEventHandler()
	err = store.LoadFromVersion(ctx, []eventstore.VersionQuery{
		{
			GroupID:     aggregateID1Path.GroupID,
			AggregateID: aggregateID1Path.AggregateID,
			Version:     getEvents(0, 6, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp)[2].Version(),
		},
	}, eh3)
	require.NoError(t, err)
	// loads the snapshot
	require.Equal(t, getEvents(5, 1, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp+5), eh2.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])

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
	require.Equal(t, getEvents(5, 1, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp+5), eh4.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])
	require.Equal(t, getEvents(3, 1, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp+3), eh4.events[aggregateID2Path.GroupID][aggregateID2Path.AggregateID])

	t.Log("load multiple aggregates by groupId")
	eh5 := NewMockEventHandler()
	err = store.LoadFromSnapshot(ctx, []eventstore.SnapshotQuery{
		{
			GroupID: aggregateID1Path.GroupID,
		},
	}, eh5)
	require.NoError(t, err)
	require.Equal(t,
		getEvents(5, 1, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp+5),
		eh5.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])
	require.Equal(t,
		getEvents(3, 1, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp+3),
		eh5.events[aggregateID2Path.GroupID][aggregateID2Path.AggregateID])

	t.Log("load multiple aggregates by all")
	eh6 := NewMockEventHandler()
	saveStatus, err = store.Save(ctx, getEvents(0, 6, aggregateID3Path.GroupID, aggregateID3Path.AggregateID, timestamp)[0])
	require.NoError(t, err)
	require.Equal(t, eventstore.Ok, saveStatus)
	err = store.LoadFromSnapshot(ctx, []eventstore.SnapshotQuery{{GroupID: aggregateID1Path.GroupID}, {GroupID: aggregateID2Path.GroupID}, {GroupID: aggregateID3Path.GroupID}}, eh6)
	require.NoError(t, err)
	require.Equal(t, getEvents(5, 1, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp+5),
		eh6.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])
	require.Equal(t,
		getEvents(3, 1, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp+3),
		eh6.events[aggregateID2Path.GroupID][aggregateID2Path.AggregateID])
	require.Equal(t, []eventstore.Event{
		getEvents(0, 1, aggregateID3Path.GroupID, aggregateID3Path.AggregateID, timestamp)[0],
	}, eh6.events[aggregateID3Path.GroupID][aggregateID3Path.AggregateID])

	t.Log("test projection all")
	model := NewMockEventHandler()
	p := eventstore.NewProjection(store, func(context.Context, string, string) (eventstore.Model, error) { return model, nil }, nil)

	err = p.Project(ctx, []eventstore.SnapshotQuery{{GroupID: aggregateID1Path.GroupID}, {GroupID: aggregateID2Path.GroupID}, {GroupID: aggregateID3Path.GroupID}})
	require.NoError(t, err)
	require.Equal(t, getEvents(5, 1, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp+5), model.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])
	require.Equal(t, getEvents(3, 1, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp+3), model.events[aggregateID2Path.GroupID][aggregateID2Path.AggregateID])
	require.Equal(t, []eventstore.Event{
		getEvents(0, 6, aggregateID3Path.GroupID, aggregateID3Path.AggregateID, timestamp)[0],
	}, model.events[aggregateID3Path.GroupID][aggregateID3Path.AggregateID])

	t.Log("test projection group")
	model1 := NewMockEventHandler()
	p = eventstore.NewProjection(store, func(context.Context, string, string) (eventstore.Model, error) { return model1, nil }, nil)

	err = p.Project(ctx, []eventstore.SnapshotQuery{{GroupID: aggregateID1Path.GroupID}})
	require.NoError(t, err)
	require.Equal(t, getEvents(5, 1, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp+5), model1.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])
	require.Equal(t, getEvents(3, 1, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp+3), model1.events[aggregateID2Path.GroupID][aggregateID2Path.AggregateID])

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
	require.Equal(t, getEvents(3, 1, aggregateID2Path.GroupID, aggregateID2Path.AggregateID, timestamp+3), model2.events[aggregateID2Path.GroupID][aggregateID2Path.AggregateID])

	eh10 := NewMockEventHandler()
	err = store.LoadFromVersion(ctx, []eventstore.VersionQuery{
		{
			GroupID:     aggregateID1Path.GroupID,
			AggregateID: aggregateID1Path.AggregateID,
		},
	}, eh10)
	require.NoError(t, err)
	require.Equal(t, getEvents(5, 1, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, timestamp+5), eh10.events[aggregateID1Path.GroupID][aggregateID1Path.AggregateID])
}
