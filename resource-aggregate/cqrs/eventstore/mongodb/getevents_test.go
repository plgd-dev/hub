package mongodb_test

import (
	"context"
	"errors"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/kit/v2/strings"
	"github.com/stretchr/testify/require"
)

type dummyEventHandler struct {
}

func (eh *dummyEventHandler) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return errors.New("cannot determine type of event")
		}
	}
	return nil
}

const (
	getEventsDeviceCount   = 10
	getEventsResourceCount = 2000
)

func addEventsForGetEventsToDB(t *testing.T, ctx context.Context, store *mongodb.EventStore) int {
	const eventCount = 100000
	var resourceVersion [getEventsResourceCount]uint64
	var resourceTimestamp [getEventsResourceCount]int64
	var resourceEvents [getEventsResourceCount][]eventstore.Event
	for i := 0; i < eventCount; i++ {
		deviceIndex := i % getEventsDeviceCount
		resourceIndex := i % getEventsResourceCount
		if i < getEventsResourceCount {
			resourceTimestamp[i] = int64((eventCount / getEventsResourceCount) * i)
		}

		resourceEvents[resourceIndex] = append(resourceEvents[resourceIndex], test.MockEvent{
			VersionI:     resourceVersion[resourceIndex],
			EventTypeI:   "testType",
			IsSnapshotI:  false,
			AggregateIDI: "resource" + strconv.Itoa(resourceIndex),
			GroupIDI:     "device" + strconv.Itoa(deviceIndex),
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

func getEventsByTimestamp(t *testing.T, ctx context.Context, store *mongodb.EventStore, queries []eventstore.GetEventsQuery, timestamp int64) {
	err := store.GetEvents(ctx, queries, timestamp, &dummyEventHandler{})
	require.NoError(t, err)
}

type getEventsQueryGenerator func() []eventstore.GetEventsQuery

type runGetEventsConfig struct {
	iterations int
	queries    []eventstore.GetEventsQuery
	generator  getEventsQueryGenerator
}

func runGetEvents(t *testing.T, cfg runGetEventsConfig) {
	logger := log.NewLogger(log.MakeDefaultConfig())

	ctx := context.Background()
	store, err := NewTestEventStore(ctx, logger)
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() {
		t.Log("clearing db")
		err = store.Clear(ctx)
		require.NoError(t, err)
		err := store.Close(ctx)
		require.NoError(t, err)
	}()

	eventCount := addEventsForGetEventsToDB(t, ctx, store)

	rand.Seed(time.Now().UnixNano())
	start := time.Now()
	for i := 0; i < cfg.iterations; i++ {
		if cfg.queries != nil {
			getEventsByTimestamp(t, ctx, store, cfg.queries, int64(rand.Intn(eventCount+1)))
		} else {
			getEventsByTimestamp(t, ctx, store, cfg.generator(), int64(rand.Intn(eventCount+1)))
		}
	}
	end := time.Now()
	elapsed := end.Sub(start)
	t.Logf("elapsed: %v", elapsed)
}

func TestGetEventsByTimestamp(t *testing.T) {
	runGetEvents(t, runGetEventsConfig{
		iterations: 2000,
		queries: []eventstore.GetEventsQuery{
			{
				GroupID: "device1",
			},
		},
	})
}

func TestGetDeviceEventsByTimestamp(t *testing.T) {
	runGetEvents(t, runGetEventsConfig{
		iterations: 200,
		queries: []eventstore.GetEventsQuery{
			{
				GroupID: "device0",
			}, {
				GroupID: "device2",
			}, {
				GroupID: "device4",
			}, {
				GroupID: "device6",
			}, {
				GroupID: "device8",
			},
		},
	})
}

func TestGetResourceEventsByTimestamp(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	runGetEvents(t, runGetEventsConfig{
		iterations: 5000,
		generator: func() []eventstore.GetEventsQuery {
			resourceIndex := rand.Intn(getEventsResourceCount + 1)
			deviceIndex := resourceIndex % getEventsDeviceCount
			return []eventstore.GetEventsQuery{
				{
					GroupID:     "device" + strconv.Itoa(deviceIndex),
					AggregateID: "resource" + strconv.Itoa(resourceIndex),
				},
			}
		},
	})
}

func TestGetResourcesEventsByTimestamp(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	runGetEvents(t, runGetEventsConfig{
		iterations: 5000,
		generator: func() []eventstore.GetEventsQuery {
			queries := make([]eventstore.GetEventsQuery, 5)
			for i := range queries {
				resourceIndex := rand.Intn(getEventsResourceCount + 1)
				deviceIndex := resourceIndex % getEventsDeviceCount
				queries[i].GroupID = "device" + strconv.Itoa(deviceIndex)
				queries[i].AggregateID = "resource" + strconv.Itoa(resourceIndex)
			}
			return queries
		},
	})
}

func Test_getNormalizedGetEventsFilter(t *testing.T) {
	const groupID1 = "groupID1"
	const aggregateID1 = "aggregateID1"
	const aggregateID2 = "aggregateID2"

	type args struct {
		queries []eventstore.GetEventsQuery
	}
	tests := []struct {
		name string
		args args
		want mongodb.DeviceIdFilter
	}{
		{
			name: "Remove duplicates",
			args: args{
				queries: []eventstore.GetEventsQuery{
					{
						GroupID:     groupID1,
						AggregateID: aggregateID1,
					},
					{
						GroupID:     groupID1,
						AggregateID: aggregateID2,
					},
					{
						GroupID:     groupID1,
						AggregateID: aggregateID1,
					},
				},
			},
			want: mongodb.DeviceIdFilter{
				All: false,
				DeviceIds: map[string]mongodb.ResourceIdFilter{
					groupID1: {
						All:         false,
						ResourceIds: strings.MakeSet(aggregateID1, aggregateID2),
					},
				},
			},
		},
		{
			name: "Absorb aggregate ids",
			args: args{
				queries: []eventstore.GetEventsQuery{
					{
						GroupID:     groupID1,
						AggregateID: aggregateID1,
					},
					{
						GroupID:     groupID1,
						AggregateID: aggregateID2,
					},
					{
						GroupID: groupID1,
					},
				},
			},
			want: mongodb.DeviceIdFilter{
				All: false,
				DeviceIds: map[string]mongodb.ResourceIdFilter{
					groupID1: {
						All: true,
					},
				},
			},
		},
		{
			name: "Absorb group ids",
			args: args{
				queries: []eventstore.GetEventsQuery{
					{
						GroupID:     groupID1,
						AggregateID: aggregateID1,
					},
					{
						GroupID:     groupID1,
						AggregateID: aggregateID2,
					},
					{
						GroupID: "",
					},
				},
			},
			want: mongodb.DeviceIdFilter{
				All: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mongodb.GetNormalizedGetEventsFilter(tt.args.queries)
			require.Equal(t, got, tt.want)
		})
	}
}
